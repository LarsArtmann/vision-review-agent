package reviewed_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	reviewed "github.com/larsartmann/vision-review-agent/internal/reviewd"
)

// countingRunner wraps a pipeline and counts passes, so specs can observe
// that the daemon loop keeps ticking.
type countingRunner struct {
	inner *reviewed.Pipeline
	mu    sync.Mutex
	count int
}

func (c *countingRunner) Pass(ctx context.Context, projects map[string][]string) (reviewed.PassResult, error) {
	c.mu.Lock()
	c.count++
	c.mu.Unlock()

	return c.inner.Pass(ctx, projects)
}

func (c *countingRunner) passes() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.count
}

// daemonMinPasses is the pass count (first immediate + one tick) the daemon
// spec waits for before cancelling.
const daemonMinPasses = 2

// daemonTick waits long enough for at least one interval to elapse.
const daemonTick = 5 * time.Millisecond

var _ = Describe("Daemon", func() {
	var (
		store      *reviewed.Store
		projects   map[string][]string
		cancel     context.CancelFunc
		stopped    chan error
		runner     *countingRunner
		shotsDir   string
		dataDir    string
		reviewsDir string
	)

	startDaemon := func(interval time.Duration) *reviewed.Daemon {
		model := &stubLanguageModel{markdown: "## Review\nFine.\n\n**Score: 7/10**"}
		Expect(writeShotPNG(filepath.Join(shotsDir, "Home--dark--desktop.png"))).To(Succeed())

		reviewer, err := reviewed.NewReviewer(model, "stub-model", 0)
		Expect(err).NotTo(HaveOccurred())

		pipeline, err := reviewed.NewPipeline(
			reviewer,
			store,
			reviewed.NewBlobStore(dataDir),
			reviewed.NewWriter(reviewsDir),
			nil,
		)
		Expect(err).NotTo(HaveOccurred())

		runner = &countingRunner{inner: pipeline}

		daemon, daemonErr := reviewed.NewDaemon(runner, projects, interval, slog.Default())
		Expect(daemonErr).NotTo(HaveOccurred())

		var ctx context.Context

		ctx, cancel = context.WithCancel(context.Background())
		stopped = make(chan error, 1)

		go func() { stopped <- daemon.Run(ctx) }()

		return daemon
	}

	BeforeEach(func() {
		var err error

		shotsDir, err = os.MkdirTemp("", "reviewd-daemon-shots-*")
		Expect(err).NotTo(HaveOccurred())

		dataDir, err = os.MkdirTemp("", "reviewd-daemon-data-*")
		Expect(err).NotTo(HaveOccurred())

		reviewsDir, err = os.MkdirTemp("", "reviewd-daemon-reviews-*")
		Expect(err).NotTo(HaveOccurred())

		store, err = reviewed.OpenStore(filepath.Join(dataDir, "events.db"), slog.Default())
		Expect(err).NotTo(HaveOccurred())

		projects = map[string][]string{"myapp": {filepath.Join(shotsDir, "*.png")}}
	})

	AfterEach(func() {
		if cancel != nil {
			cancel()
		}

		if stopped != nil {
			select {
			case err := <-stopped:
				Expect(err).NotTo(HaveOccurred())
			default:
			}
		}

		Expect(store.Close()).To(Succeed())
		Expect(os.RemoveAll(shotsDir)).To(Succeed())
		Expect(os.RemoveAll(dataDir)).To(Succeed())
		Expect(os.RemoveAll(reviewsDir)).To(Succeed())
	})

	Context("when constructed with invalid wiring", func() {
		It("rejects a nil pass runner", func() {
			_, err := reviewed.NewDaemon(nil, projects, time.Minute, nil)
			Expect(err).To(MatchError(reviewed.ErrNoPipeline))
		})

		It("rejects an empty project set", func() {
			_, err := reviewed.NewDaemon(runner, map[string][]string{}, time.Minute, nil)
			Expect(err).To(MatchError(reviewed.ErrInvalidProjects))
		})

		It("rejects a non-positive interval", func() {
			_, err := reviewed.NewDaemon(runner, projects, 0, nil)
			Expect(err).To(MatchError(reviewed.ErrInvalidInterval))
		})
	})

	It("reviews immediately and then on every tick, and stops cleanly", func() {
		startDaemon(daemonTick)

		Eventually(runner.passes, "5s", "5ms").Should(BeNumerically(">=", daemonMinPasses))

		state, _, err := store.LoadView(context.Background(), "myapp", homeViewKey())
		Expect(err).NotTo(HaveOccurred())
		Expect(state.Reviews).To(Equal(1))
		Expect(state.Captures).To(Equal(1))

		index := readFileOrFail(filepath.Join(reviewsDir, "myapp", "INDEX.md"))
		Expect(index).To(ContainSubstring("7/10"))

		cancel()

		Eventually(stopped, "5s").Should(Receive(BeNil()))
	})
})
