package reviewed

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestPipelineConcurrentPassesStayRaceClean pins the concurrency contract:
// the daemon's ticker serializes passes (one pass finishes before the next
// starts), but nothing structurally prevents a caller from overlapping two
// Pass invocations. This test runs concurrent passes over ONE pipeline
// (shared store, blob store, writer) under the race detector and then folds
// the journal. The observed contract, now pinned: memory races surface as
// test failures; the store's optimistic version control REJECTS interleaved
// writes with version-conflict errors (view work fails, nothing corrupts);
// and the journal stays readable and foldable throughout. In other words:
// overlapping passes lose work, never integrity — and the daemon's
// serialization is what prevents the lost work.
func TestPipelineConcurrentPassesStayRaceClean(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	dataDir := filepath.Join(base, "data")
	reviewsDir := filepath.Join(base, "reviews")
	shots := filepath.Join(base, "shots")

	writePNGFile(t, filepath.Join(shots, "Home--dark--desktop.png"), 0x40)
	writePNGFile(t, filepath.Join(shots, "Settings--light--desktop.png"), 0xc0)

	reviewer, err := NewReviewer(newMockReviewModel("## Review\nFine.\n\n**Score: 6/10**"), NewModelID("stub-model"), 0)
	if err != nil {
		t.Fatalf("build reviewer: %v", err)
	}

	store, err := OpenStore(JournalPath(dataDir), slog.Default())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	defer closeStoreQuietly(store)

	pipeline, err := NewPipeline(reviewer, store, NewBlobStore(dataDir), NewWriter(reviewsDir), slog.Default())
	if err != nil {
		t.Fatalf("build pipeline: %v", err)
	}

	projects := map[string][]string{"app": {shots + "/*.png"}}

	const (
		goroutines = 2
		passesEach = 5
	)

	var (
		wg           sync.WaitGroup
		conflicts    sync.Map
		nonConflicts sync.Map
	)

	for range goroutines {
		wg.Add(1)

		wg.Go(func() {
			for range passesEach {
				if _, passErr := pipeline.Pass(context.Background(), projects); passErr != nil {
					if strings.Contains(passErr.Error(), "version_conflict") {
						conflicts.Store(passErr.Error(), struct{}{})
					} else {
						nonConflicts.Store(passErr.Error(), struct{}{})
					}
				}
			}
		})
	}

	wg.Wait()

	nonConflicts.Range(func(key, _ any) bool {
		t.Errorf("concurrent pass failed with a non-conflict error: %v", key)

		return true
	})

	// The interleaved journal must still open, read, and fold cleanly —
	// concurrency may duplicate reviews but must never corrupt the store.
	events, err := store.AllEvents(context.Background())
	if err != nil {
		t.Fatalf("read interleaved journal: %v", err)
	}

	if _, err := VerifyJournalEvents(events); err != nil {
		t.Fatalf("fold interleaved journal: %v", err)
	}

	conflictCount := 0

	conflicts.Range(func(_, _ any) bool {
		conflictCount++

		return true
	})

	t.Logf("%d interleaved write conflicts rejected by the store's version check", conflictCount)
}
