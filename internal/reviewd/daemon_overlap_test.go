package reviewed

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// slowTrackingRunner sleeps inside every pass and records the maximum
// number of concurrent Pass invocations, so a single test can prove the
// daemon loop never overlaps passes even when the interval is far shorter
// than a pass.
type slowTrackingRunner struct {
	inFlight atomic.Int32
	maxSeen  atomic.Int32
}

func (s *slowTrackingRunner) Pass(_ context.Context, _ map[string][]string) (PassResult, error) {
	current := s.inFlight.Add(1)

	for {
		max := s.maxSeen.Load()
		if current <= max || s.maxSeen.CompareAndSwap(max, current) {
			break
		}
	}

	time.Sleep(30 * time.Millisecond)

	s.inFlight.Add(-1)

	return PassResult{}, nil
}

// TestDaemonPassesNeverOverlap pins the design guarantee that passes are
// strictly sequential: a daemon whose interval (5ms) is much shorter than
// its pass (30ms) must still never have two passes in flight at once.
func TestDaemonPassesNeverOverlap(t *testing.T) {
	t.Parallel()

	runner := &slowTrackingRunner{}

	daemon, err := NewDaemon(
		runner,
		map[string][]string{"app": {"/nonexistent/*.png"}},
		5*time.Millisecond,
		slog.Default(),
	)
	if err != nil {
		t.Fatalf("new daemon: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if err := daemon.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}

	if runner.maxSeen.Load() > 1 {
		t.Fatalf("max concurrent passes = %d, want 1 (passes must never overlap)", runner.maxSeen.Load())
	}
}
