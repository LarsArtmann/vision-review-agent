package reviewed

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCompareManuallyRecordsEventAndWritesMarkdown(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	beforePath := writeComparePNG(t, dir, "before", scanTestPNG)
	afterPath := writeComparePNG(t, dir, "Home--dark--desktop", changedScanPNG())

	model := newMockReviewModel("## Diff\nSpacing improved.\n\n**Score: 9/10**")

	reviewer, err := NewReviewer(model, "test-model", time.Minute)
	if err != nil {
		t.Fatalf("NewReviewer: %v", err)
	}

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

	writer := NewWriter(reviewsDir)

	pipeline, err := NewPipeline(reviewer, store, NewBlobStore(dataDir), writer, nil)
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}

	if err := pipeline.CompareManually(t.Context(), "myapp", beforePath, afterPath); err != nil {
		t.Fatalf("CompareManually: %v", err)
	}

	viewKey := ViewKey{Page: "Home", Theme: "dark", Viewport: "desktop"}

	state, _, err := store.LoadView(t.Context(), "myapp", viewKey)
	if err != nil {
		t.Fatalf("LoadView: %v", err)
	}

	if state.Comparisons != 1 {
		t.Fatalf("comparisons = %d, want 1", state.Comparisons)
	}

	if state.Captures != 0 {
		t.Fatalf("manual compare must not record captures, got %d", state.Captures)
	}

	events, err := store.AllEvents(t.Context())
	if err != nil {
		t.Fatalf("AllEvents: %v", err)
	}

	if len(events) != 1 || string(events[0].Type()) != EventViewCompared {
		t.Fatalf("want exactly one view.compared event, got %d events", len(events))
	}

	pattern := filepath.Join(reviewsDir, "myapp", "comparisons", "*_Home--dark--desktop.md")

	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) != 1 {
		t.Fatalf("comparison files = %v (err %v), want exactly 1", matches, err)
	}

	content, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read comparison: %v", err)
	}

	if !strings.Contains(string(content), "## Diff") || !strings.Contains(string(content), "9/10") {
		t.Fatalf("comparison markdown missing model output:\n%s", content)
	}

	if model.calls() != 1 {
		t.Fatalf("model calls = %d, want 1", model.calls())
	}
}

func TestCompareManuallyEmptyAfterName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	beforePath := writeComparePNG(t, dir, "before", scanTestPNG)
	afterPath := filepath.Join(dir, ".png")

	reviewer, err := NewReviewer(newMockReviewModel("x"), "test-model", time.Minute)
	if err != nil {
		t.Fatalf("NewReviewer: %v", err)
	}

	store, err := OpenStore(filepath.Join(dir, "events.db"), slog.Default())
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}

	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close store: %v", closeErr)
		}
	}()

	pipeline, err := NewPipeline(reviewer, store, NewBlobStore(dir), NewWriter(dir), nil)
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}

	err = pipeline.CompareManually(t.Context(), "myapp", beforePath, afterPath)
	if err == nil {
		t.Fatal("want error for unparseable after file name")
	}
}

func writeComparePNG(t *testing.T, dir string, name string, png []byte) string {
	t.Helper()

	path := filepath.Join(dir, name+".png")
	if err := os.WriteFile(path, png, 0o644); err != nil {
		t.Fatalf("write png %s: %v", path, err)
	}

	return path
}

func changedScanPNG() []byte {
	changed := make([]byte, len(scanTestPNG))
	copy(changed, scanTestPNG)

	const idatPixelOffset = 45

	changed[idatPixelOffset] ^= 0xFF

	return changed
}
