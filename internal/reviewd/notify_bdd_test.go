package reviewed_test

import (
	"context"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	reviewed "github.com/larsartmann/vision-review-agent/internal/reviewd"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeNotifier records daemon liveness traffic for specs.
type fakeNotifier struct {
	mu       sync.Mutex
	ready    int
	pings    int
	interval time.Duration
}

func (f *fakeNotifier) Ready() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.ready++

	return nil
}

func (f *fakeNotifier) WatchdogPing() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.pings++

	return nil
}

func (f *fakeNotifier) WatchdogInterval() time.Duration { return f.interval }

func (f *fakeNotifier) counts() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.ready, f.pings
}

// instantPassRunner completes passes immediately so the daemon loop itself
// contributes no timing noise to the heartbeat specs.
type instantPassRunner struct{}

func (instantPassRunner) Pass(_ context.Context, _ map[string][]string) (reviewed.PassResult, error) {
	return reviewed.PassResult{}, nil
}

var _ = Describe("ServiceNotifier", func() {
	Describe("under systemd (NOTIFY_SOCKET set)", func() {
		It("sends READY=1 on the notify socket", func() {
			socketPath := filepath.Join(GinkgoT().TempDir(), "notify.sock")

			receiver, err := net.ListenPacket("unixgram", socketPath)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() { _ = receiver.Close() })

			Expect(os.Setenv("NOTIFY_SOCKET", socketPath)).To(Succeed())
			DeferCleanup(func() { _ = os.Unsetenv("NOTIFY_SOCKET") })

			notifier := reviewed.NewServiceNotifier()
			Expect(notifier.Ready()).To(Succeed())

			buffer := make([]byte, 64)

			Expect(receiver.SetReadDeadline(time.Now().Add(5 * time.Second))).To(Succeed())

			n, _, readErr := receiver.ReadFrom(buffer)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(string(buffer[:n])).To(Equal("READY=1"))
		})

		It("derives the ping cadence from WATCHDOG_USEC at half the budget", func() {
			Expect(os.Setenv("NOTIFY_SOCKET", "/nonexistent/notify.sock")).To(Succeed())
			DeferCleanup(func() { _ = os.Unsetenv("NOTIFY_SOCKET") })

			Expect(os.Setenv("WATCHDOG_USEC", "30000000")).To(Succeed())
			DeferCleanup(func() { _ = os.Unsetenv("WATCHDOG_USEC") })

			Expect(reviewed.NewServiceNotifier().WatchdogInterval()).To(Equal(15 * time.Second))
		})
	})

	Describe("outside systemd", func() {
		It("is inert: no socket, no watchdog cadence", func() {
			Expect(os.Unsetenv("NOTIFY_SOCKET")).To(Succeed())
			Expect(os.Unsetenv("WATCHDOG_USEC")).To(Succeed())

			notifier := reviewed.NewServiceNotifier()
			Expect(notifier.Ready()).To(Succeed())
			Expect(notifier.WatchdogInterval()).To(BeZero())
		})
	})
})

var _ = Describe("Daemon watchdog heartbeat", func() {
	It("announces READY once and pings while running, then stops with the context", func() {
		notifier := &fakeNotifier{interval: 5 * time.Millisecond}

		ctx, cancel := context.WithCancel(context.Background())
		daemon, err := reviewed.NewDaemon(
			instantPassRunner{},
			map[string][]string{"proj": {"/nonexistent/*.png"}},
			250*time.Millisecond,
			slog.Default(),
			reviewed.WithServiceNotifier(notifier),
		)
		Expect(err).NotTo(HaveOccurred())

		runDone := make(chan error, 1)
		go func() { runDone <- daemon.Run(ctx) }()

		Eventually(func() int {
			_, pings := notifier.counts()

			return pings
		}, "2s", "5ms").Should(BeNumerically(">=", 2))

		ready, _ := notifier.counts()
		Expect(ready).To(Equal(1))

		cancel()
		Eventually(runDone, "2s").Should(Receive(Succeed()))

		// READY is sent exactly once at startup and never again; ping count
		// is allowed one late tick (a tick already queued when the context
		// fired), which is why the stop check pins ready, not pings.
		Consistently(func() int {
			ready, _ := notifier.counts()

			return ready
		}, "150ms", "25ms").Should(Equal(1))
	})
})
