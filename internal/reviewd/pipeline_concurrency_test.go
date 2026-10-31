package reviewed

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"
)

// TestPipelinePassHandoffStaysFoldable pins the concurrency contract from
// the safe side. The daemon's ticker serializes passes; the store's
// optimistic version control plus bbolt's single-writer lock make an
// OVERLAP lose work (version_conflict failures) or block (unbounded journal
// write lock) — corruption is impossible, but overlapping callers are not
// supported and are NOT exercised here (an overlap probe hung on the writer
// lock for the full 600s test timeout on 2026-10-31; that hang IS the
// documented reason the ticker exists). What this test pins instead: the
// same pipeline handed between goroutines — every pass fully serialized —
// keeps the journal readable and foldable, and the race detector sees the
// cross-goroutine handoff of every shared dependency.
func TestPipelinePassHandoffStaysFoldable(t *testing.T) {
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

	// Three goroutines own the pipeline strictly one at a time; the channel
	// handoff gives the race detector real cross-goroutine edges on the
	// shared store, blobs, and writer without ever overlapping passes.
	const (
		handoffs   = 3
		passesEach = 2
	)

	for range handoffs {
		done := make(chan error, 1)

		go func() {
			var passErr error

			for range passesEach {
				if _, err := pipeline.Pass(context.Background(), projects); err != nil {
					passErr = err

					break
				}
			}

			done <- passErr
		}()

		if err := <-done; err != nil {
			t.Fatalf("serialized pass: %v", err)
		}
	}

	events, err := store.AllEvents(context.Background())
	if err != nil {
		t.Fatalf("read journal after handoffs: %v", err)
	}

	if _, err := VerifyJournalEvents(events); err != nil {
		t.Fatalf("fold journal after handoffs: %v", err)
	}
}
