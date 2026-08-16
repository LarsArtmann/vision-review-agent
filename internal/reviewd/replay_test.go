package reviewed

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// snapshotFiles returns the byte content of every file under root, keyed by
// its path relative to root. Paths are collected during the walk and read
// afterwards, so no filesystem operation races the walk itself.
func snapshotFiles(t *testing.T, root string) map[string]string {
	t.Helper()

	relativePaths := make([]string, 0)

	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}

		if entry.IsDir() {
			return nil
		}

		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return fmt.Errorf("rel %s against %s: %w", path, root, relErr)
		}

		relativePaths = append(relativePaths, relative)

		return nil
	})
	if walkErr != nil {
		t.Fatalf("snapshot %s: %v", root, walkErr)
	}

	snapshot := make(map[string]string, len(relativePaths))

	for _, relative := range relativePaths {
		data, readErr := os.ReadFile(filepath.Join(root, relative))
		if readErr != nil {
			t.Fatalf("read %s: %v", relative, readErr)
		}

		snapshot[relative] = string(data)
	}

	return snapshot
}

func TestReplayRebuildsWipedReviewsDirByteIdentical(t *testing.T) {
	t.Parallel()

	shotsDir := t.TempDir()
	dataDir := t.TempDir()
	reviewsDir := t.TempDir()

	shotPath := filepath.Join(shotsDir, "Home--dark--desktop.png")

	if err := os.WriteFile(shotPath, scanTestPNG, 0o644); err != nil {
		t.Fatalf("write shot: %v", err)
	}

	store, err := OpenStore(filepath.Join(dataDir, "events.db"), slog.Default())
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}

	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close store: %v", closeErr)
		}
	}()

	reviewer, err := NewReviewer(newMockReviewModel("## Review\nFine.\n\n**Score: 7/10**"), "test-model", time.Minute)
	if err != nil {
		t.Fatalf("NewReviewer: %v", err)
	}

	pipeline, err := NewPipeline(reviewer, store, NewBlobStore(dataDir), NewWriter(reviewsDir), nil)
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}

	projects := map[string][]string{"myapp": {filepath.Join(shotsDir, "*.png")}}

	if _, err := pipeline.Pass(t.Context(), projects); err != nil {
		t.Fatalf("first pass: %v", err)
	}

	if err := os.WriteFile(shotPath, changedScanPNG(), 0o644); err != nil {
		t.Fatalf("write changed shot: %v", err)
	}

	if _, err := pipeline.Pass(t.Context(), projects); err != nil {
		t.Fatalf("second pass: %v", err)
	}

	before := snapshotFiles(t, reviewsDir)

	if len(before) < replayMinFilesAfterPasses {
		t.Fatalf("expected reviews files after two passes, got %d: %v", len(before), before)
	}

	if err := os.RemoveAll(reviewsDir); err != nil {
		t.Fatalf("wipe reviews dir: %v", err)
	}

	result, replayErr := Replay(t.Context(), store, NewWriter(reviewsDir))
	if replayErr != nil {
		t.Fatalf("Replay: %v", replayErr)
	}

	if result.Projects != 1 || result.Views != 1 || result.Reviews != 2 || result.Comparisons != 1 {
		t.Fatalf("replay result = %+v, want 1 project, 1 view, 2 reviews, 1 comparison", result)
	}

	after := snapshotFiles(t, reviewsDir)

	if len(after) != len(before) {
		t.Fatalf("replayed file count = %d, want %d (before %v, after %v)", len(after), len(before), before, after)
	}

	for path, want := range before {
		if got := after[path]; got != want {
			t.Fatalf("replayed %s differs:\n--- pass ---\n%s\n--- replay ---\n%s", path, want, got)
		}
	}
}

// replayMinFilesAfterPasses is the file count two passes must produce:
// INDEX, one view review, and one comparison.
const replayMinFilesAfterPasses = 3

func TestReplayEmptyJournalWritesNothing(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	reviewsDir := t.TempDir()

	store, err := OpenStore(filepath.Join(dataDir, "events.db"), slog.Default())
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}

	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close store: %v", closeErr)
		}
	}()

	result, replayErr := Replay(t.Context(), store, NewWriter(reviewsDir))
	if replayErr != nil {
		t.Fatalf("Replay on empty journal: %v", replayErr)
	}

	if result.Projects != 0 || result.Views != 0 || result.Reviews != 0 || result.Comparisons != 0 {
		t.Fatalf("replay result = %+v, want all zero", result)
	}

	entries, readErr := os.ReadDir(reviewsDir)
	if readErr != nil {
		t.Fatalf("read reviews dir: %v", readErr)
	}

	if len(entries) != 0 {
		t.Fatalf("replay of empty journal wrote %d entries, want 0", len(entries))
	}
}

func TestSummarizeEventsIncludesScoresAndHashes(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	store, err := OpenStore(filepath.Join(dataDir, "events.db"), slog.Default())
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}

	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close store: %v", closeErr)
		}
	}()

	viewKey := ViewKey{Page: "Home", Theme: "dark", Viewport: "desktop"}

	ctx := t.Context()

	capturedAt := time.Date(2026, 8, 16, 20, 0, 0, 0, time.UTC)

	if err := store.RecordCapture(ctx, "myapp", viewKey, Captured{
		SourcePath: "/shots/Home--dark--desktop.png",
		BlobPath:   "blobs/aa",
		SHA256:     strings.Repeat("a", 64),
		CapturedAt: capturedAt,
	}); err != nil {
		t.Fatalf("RecordCapture: %v", err)
	}

	if err := store.RecordReview(ctx, "myapp", viewKey, Reviewed{
		SHA256:     strings.Repeat("a", 64),
		Model:      "test-model",
		Markdown:   "fine",
		Score:      7,
		ReviewedAt: capturedAt.Add(time.Minute),
	}); err != nil {
		t.Fatalf("RecordReview: %v", err)
	}

	events, err := store.AllEvents(ctx)
	if err != nil {
		t.Fatalf("AllEvents: %v", err)
	}

	summaries := SummarizeEvents(events)

	if len(summaries) != 2 {
		t.Fatalf("summaries = %d, want 2", len(summaries))
	}

	capture, review := summaries[0], summaries[1]

	if capture.Project != "myapp" || capture.ViewKey.String() != "Home--dark--desktop" {
		t.Fatalf("capture summary address = %s:%s", capture.Project, capture.ViewKey)
	}

	if capture.Type != EventViewCaptured || !strings.Contains(capture.Detail, "sha="+strings.Repeat("a", 12)) {
		t.Fatalf("capture summary = %+v", capture)
	}

	if review.Type != EventViewReviewed || !strings.Contains(review.Detail, "score=7/10") {
		t.Fatalf("review summary = %+v", review)
	}

	if review.Version <= capture.Version {
		t.Fatalf("review version %d should exceed capture version %d", review.Version, capture.Version)
	}
}
