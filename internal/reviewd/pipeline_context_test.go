package reviewed

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"charm.land/fantasy"
)

// cancelOnGenerateModel cancels the pass context during its first Generate
// call, then delegates to the wrapped mock, so the pipeline sees a context
// that dies mid-pass.
type cancelOnGenerateModel struct {
	*mockReviewModel

	cancel context.CancelFunc
	once   sync.Once
}

func (m *cancelOnGenerateModel) Generate(ctx context.Context, in fantasy.Call) (*fantasy.Response, error) {
	m.once.Do(m.cancel)

	return m.mockReviewModel.Generate(ctx, in)
}

func newCancelledPassPipeline(t *testing.T, model fantasy.LanguageModel) (*Pipeline, string, string) {
	t.Helper()

	shots := filepath.Join(t.TempDir(), "shots")
	if err := os.MkdirAll(shots, 0o750); err != nil {
		t.Fatalf("mkdir shots: %v", err)
	}

	writeComparePNG(t, shots, "Home--dark--desktop", scanTestPNG)
	writeComparePNG(t, shots, "Settings--dark--desktop", scanTestPNG)

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

	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close store: %v", closeErr)
		}
	})

	pipeline, err := NewPipeline(reviewer, store, NewBlobStore(dataDir), NewWriter(reviewsDir), nil)
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}

	return pipeline, reviewsDir, filepath.Join(shots, "*.png")
}

func TestPassPreCancelledContextStopsBeforeAnyProject(t *testing.T) {
	t.Parallel()

	model := newMockReviewModel("## Review\nfine\n\n**Score: 8/10**")
	pipeline, reviewsDir, glob := newCancelledPassPipeline(t, model)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	result, err := pipeline.Pass(ctx, map[string][]string{"myapp": {glob}})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Pass error = %v, want context.Canceled", err)
	}

	if !strings.Contains(err.Error(), "skipped, pass context done") {
		t.Fatalf("Pass error should name the skip explicitly:\n%v", err)
	}

	if result.Projects != 0 || result.Reviewed != 0 {
		t.Fatalf("result = %+v, want nothing processed", result)
	}

	if model.calls() != 0 {
		t.Fatalf("model calls = %d, want 0", model.calls())
	}

	if _, statErr := os.Stat(filepath.Join(reviewsDir, "myapp", "INDEX.md")); !os.IsNotExist(statErr) {
		t.Fatalf("INDEX must not be written on a cancelled pass, stat err = %v", statErr)
	}
}

func TestPassContextCancelledMidPassSkipsRemainingViews(t *testing.T) {
	t.Parallel()

	model := &cancelOnGenerateModel{
		mockReviewModel: newMockReviewModel("## Review\nfine\n\n**Score: 8/10**"),
	}
	pipeline, reviewsDir, glob := newCancelledPassPipeline(t, model)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	model.cancel = cancel

	result, err := pipeline.Pass(ctx, map[string][]string{"myapp": {glob}})

	if err == nil || !strings.Contains(err.Error(), "skipped, pass context done") {
		t.Fatalf("Pass error = %v, want explicit skip of the remaining view", err)
	}

	if model.calls() != 1 {
		t.Fatalf("model calls = %d, want exactly 1 (second view must be skipped)", model.calls())
	}

	if result.Reviewed != 1 || result.Views != 2 {
		t.Fatalf("result = %+v, want 1 review over 2 views", result)
	}

	if _, statErr := os.Stat(filepath.Join(reviewsDir, "myapp", "INDEX.md")); !os.IsNotExist(statErr) {
		t.Fatalf("INDEX refresh must be skipped on a cancelled pass, stat err = %v", statErr)
	}
}
