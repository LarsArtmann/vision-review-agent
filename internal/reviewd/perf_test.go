package reviewed

// Journal-bound performance evidence: benchmarks for the daemon's core
// paths (Pass steady-state, Replay, raw journal read) at two scales, plus
// the numbers cited in docs/DEPS.md (snapshot revisit trigger) and the
// perf-baseline status snapshot. Benchmarks do not run in the normal test
// suite; produce the baseline with:
//
//	go test ./internal/reviewd -run '^$' -bench 'BenchmarkReplay|BenchmarkPassSteady|BenchmarkAllEvents' -benchmem -benchtime 5x

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// benchWorld is one benchmark environment: a shots tree, a seeded journal,
// and a reviews output dir.
type benchWorld struct {
	dir      string
	store    *Store
	pipeline *Pipeline
	projects map[string][]string
	views    int
}

// newBenchWorld builds a world with `views` goldens and seeds it by running
// `passes` pipeline passes, mutating every golden between passes so each
// pass appends one capture + one review per view (2*views*passes events).
func newBenchWorld(b *testing.B, dir string, views, passes int) *benchWorld {
	b.Helper()

	shotsDir := filepath.Join(dir, "shots")
	dataDir := filepath.Join(dir, "data")
	reviewsDir := filepath.Join(dir, "reviews")

	if err := os.MkdirAll(shotsDir, 0o750); err != nil {
		b.Fatalf("mkdir shots: %v", err)
	}

	shotPaths := make([]string, 0, views)

	for view := range views {
		shotPath := filepath.Join(shotsDir, fmt.Sprintf("View%03d--dark--desktop.png", view))

		if err := os.WriteFile(shotPath, scanTestPNG, 0o644); err != nil {
			b.Fatalf("write shot: %v", err)
		}

		shotPaths = append(shotPaths, shotPath)
	}

	store, err := OpenStore(filepath.Join(dataDir, "events.db"), slog.Default())
	if err != nil {
		b.Fatalf("OpenStore: %v", err)
	}

	b.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			b.Fatalf("close store: %v", closeErr)
		}
	})

	reviewer, err := NewReviewer(newMockReviewModel("## Review\nFine.\n\n**Score: 7/10**"), "bench-model", time.Minute)
	if err != nil {
		b.Fatalf("NewReviewer: %v", err)
	}

	pipeline, err := NewPipeline(reviewer, store, NewBlobStore(dataDir), NewWriter(reviewsDir), nil)
	if err != nil {
		b.Fatalf("NewPipeline: %v", err)
	}

	world := &benchWorld{
		dir:      dir,
		store:    store,
		pipeline: pipeline,
		projects: map[string][]string{"bench": {filepath.Join(shotsDir, "*.png")}},
		views:    views,
	}

	ctx := context.Background()

	for pass := range passes {
		if pass > 0 {
			// Mutate every golden so the pass re-captures and re-reviews.
			for _, shotPath := range shotPaths {
				changed := changedScanPNG()
				changed[len(changed)/2] ^= byte(pass)

				if writeErr := os.WriteFile(shotPath, changed, 0o644); writeErr != nil {
					b.Fatalf("mutate shot: %v", writeErr)
				}
			}
		}

		if _, passErr := pipeline.Pass(ctx, world.projects); passErr != nil {
			b.Fatalf("seed pass %d: %v", pass, passErr)
		}
	}

	return world
}

// BenchmarkReplay replays a 10-view, 50-pass journal (1000 events).
func BenchmarkReplay(b *testing.B) {
	world := newBenchWorld(b, b.TempDir(), 10, 50)
	writer := NewWriter(filepath.Join(world.dir, "reviews-bench"))

	b.ResetTimer()

	for range b.N {
		if _, err := Replay(context.Background(), world.store, writer); err != nil {
			b.Fatalf("replay: %v", err)
		}
	}
}

// BenchmarkReplay10kEvents replays a 100-view, 50-pass journal (10000
// events) — the F14.3/F14.4 load scale.
func BenchmarkReplay10kEvents(b *testing.B) {
	world := newBenchWorld(b, b.TempDir(), 100, 50)
	writer := NewWriter(filepath.Join(world.dir, "reviews-bench"))

	b.ResetTimer()

	for range b.N {
		if _, err := Replay(context.Background(), world.store, writer); err != nil {
			b.Fatalf("replay: %v", err)
		}
	}
}

// BenchmarkAllEvents10kEvents reads and decodes a 10k-event journal without
// rendering — Replay's dominant component.
func BenchmarkAllEvents10kEvents(b *testing.B) {
	world := newBenchWorld(b, b.TempDir(), 100, 50)

	b.ResetTimer()

	for range b.N {
		if _, err := world.store.AllEvents(context.Background()); err != nil {
			b.Fatalf("all events: %v", err)
		}
	}
}

// BenchmarkPassSteady runs passes over a fully-seen world: the daemon's
// steady-state tick (scan, hash, skip-seen everywhere).
func BenchmarkPassSteady(b *testing.B) {
	world := newBenchWorld(b, b.TempDir(), 100, 1)

	b.ResetTimer()

	for range b.N {
		if _, err := world.pipeline.Pass(context.Background(), world.projects); err != nil {
			b.Fatalf("pass: %v", err)
		}
	}
}
