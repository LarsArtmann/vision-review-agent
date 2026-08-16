package reviewed

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// Daemon dependency sentinels: NewDaemon rejects invalid wiring with these.
var (
	ErrNoPipeline       = errors.New("daemon: nil pipeline")
	ErrInvalidInterval  = errors.New("daemon: interval must be positive")
	ErrInvalidProjects  = errors.New("daemon: no projects configured")
	ErrDaemonUnexpected = errors.New("daemon: unexpected context error")
)

// PassRunner runs one review pass over the configured projects. *Pipeline
// implements it; tests and consumers can wrap or replace it.
type PassRunner interface {
	Pass(ctx context.Context, projects map[string][]string) (PassResult, error)
}

// Daemon runs review passes on a fixed interval until its context is
// cancelled. Passes never overlap: one pass finishes before the next waits.
type Daemon struct {
	pipeline PassRunner
	projects map[string][]string
	interval time.Duration
	logger   *slog.Logger
}

// NewDaemon wires the loop. A nil logger falls back to the default logger.
func NewDaemon(
	pipeline PassRunner,
	projects map[string][]string,
	interval time.Duration,
	logger *slog.Logger,
) (*Daemon, error) {
	if pipeline == nil {
		return nil, fmt.Errorf("new daemon: %w", ErrNoPipeline)
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("new daemon: %w", ErrInvalidProjects)
	}

	if interval <= 0 {
		return nil, fmt.Errorf("new daemon: %w", ErrInvalidInterval)
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &Daemon{pipeline: pipeline, projects: projects, interval: interval, logger: logger}, nil
}

// Interval reports the delay between passes.
func (d *Daemon) Interval() time.Duration {
	return d.interval
}

// Run performs one pass immediately, then repeats every interval. Pass
// failures are logged and the loop continues: a broken project or a flaky
// model must not stop the daemon. Run returns nil on graceful cancellation
// (SIGINT/SIGTERM or ctx.Done with context.Canceled).
func (d *Daemon) Run(ctx context.Context) error {
	d.runPass(ctx)

	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}

			return fmt.Errorf("%w: %w", ErrDaemonUnexpected, ctx.Err())
		case <-ticker.C:
			d.runPass(ctx)
		}
	}
}

// runPass executes one pass and logs its outcome.
func (d *Daemon) runPass(ctx context.Context) {
	result, err := d.pipeline.Pass(ctx, d.projects)
	if err != nil {
		d.logger.Warn("review pass finished with errors",
			"err", err,
			"projects", result.Projects,
			"views", result.Views,
			"captured", result.Captured,
			"reviewed", result.Reviewed)
	}

	d.logger.Info("review pass complete",
		"projects", result.Projects,
		"views", result.Views,
		"captured", result.Captured,
		"skipped", result.Skipped,
		"reviewed", result.Reviewed,
		"compared", result.Compared)
}
