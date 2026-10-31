package reviewed

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// ServiceNotifier reports daemon liveness to the service manager over the
// sd_notify protocol. Under systemd (NOTIFY_SOCKET set) it carries READY and
// WATCHDOG pings; anywhere else it is an inert no-op, so the daemon runs
// unchanged in tests and interactive sessions.
type ServiceNotifier interface {
	// Ready tells the service manager startup finished (sd_notify READY=1).
	// Under Type=notify the unit stays "activating" until this arrives.
	Ready() error

	// WatchdogPing tells the service manager the daemon is still making
	// progress (sd_notify WATCHDOG=1). With WatchdogSec configured, a missed
	// cadence restarts the unit — that is the point: a hung model call or a
	// deadlocked loop becomes a restart instead of silent rot.
	WatchdogPing() error

	// WatchdogInterval is the negotiated ping cadence: half of the manager's
	// WATCHDOG_USEC budget (the sd_notify convention). Zero means no watchdog
	// is configured and no periodic pings are needed.
	WatchdogInterval() time.Duration
}

// NewServiceNotifier returns this process's notifier: a live socket notifier
// when NOTIFY_SOCKET is set, an inert no-op otherwise.
func NewServiceNotifier() ServiceNotifier {
	socketPath, ok := os.LookupEnv("NOTIFY_SOCKET")
	if !ok || socketPath == "" {
		return noopNotifier{}
	}

	return &notifySocketNotifier{
		socketPath:     socketPath,
		watchdogPeriod: watchdogPeriodFromEnv(),
	}
}

// noopNotifier is the outside-systemd notifier: everything succeeds and
// nothing is ever sent.
type noopNotifier struct{}

func (noopNotifier) Ready() error                    { return nil }
func (noopNotifier) WatchdogPing() error             { return nil }
func (noopNotifier) WatchdogInterval() time.Duration { return 0 }

// notifySocketNotifier speaks the sd_notify datagram protocol to the socket
// named by NOTIFY_SOCKET.
type notifySocketNotifier struct {
	socketPath     string
	watchdogPeriod time.Duration
}

func (n *notifySocketNotifier) Ready() error        { return n.notify("READY=1") }
func (n *notifySocketNotifier) WatchdogPing() error { return n.notify("WATCHDOG=1") }

func (n *notifySocketNotifier) WatchdogInterval() time.Duration { return n.watchdogPeriod }

// notify writes one state payload to the notify socket. A "@"-prefixed path
// names a socket in the abstract namespace (systemd's convention) and is
// translated to Go's leading-NUL form.
func (n *notifySocketNotifier) notify(state string) error {
	address := n.socketPath
	if strings.HasPrefix(address, "@") {
		address = "\x00" + address[1:]
	}

	conn, err := (&net.Dialer{Timeout: time.Second}).DialContext(context.Background(), "unixgram", address)
	if err != nil {
		return fmt.Errorf("notify socket %s: %w", n.socketPath, err)
	}
	defer conn.Close()

	if deadlineErr := conn.SetDeadline(time.Now().Add(time.Second)); deadlineErr != nil {
		return fmt.Errorf("notify deadline: %w", deadlineErr)
	}

	if _, writeErr := conn.Write([]byte(state)); writeErr != nil {
		return fmt.Errorf("notify write %s: %w", state, writeErr)
	}

	return nil
}

// watchdogPeriodFromEnv parses WATCHDOG_USEC (microseconds) and halves it,
// per the sd_notify convention of pinging at twice the needed rate. The
// minimum of 100ms keeps a pathological WATCHDOG_USEC from busy-looping the
// heartbeat; the 24h maximum bounds garbage input (a signed parse avoids any
// uint-to-int conversion entirely).
func watchdogPeriodFromEnv() time.Duration {
	raw, err := strconv.ParseInt(os.Getenv("WATCHDOG_USEC"), 10, 64)
	if err != nil || raw <= 0 {
		return 0
	}

	const maxUseful = int64(24 * time.Hour / time.Microsecond)
	if raw > maxUseful {
		raw = maxUseful
	}

	period := time.Duration(raw) * time.Microsecond / 2
	if period < 100*time.Millisecond {
		return 100 * time.Millisecond
	}

	return period
}
