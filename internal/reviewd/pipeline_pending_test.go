package reviewed

import (
	"bytes"
	"context"
	"log/slog"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestPassPendingCountsFailedReviews proves the Pending counter is the
// greppable degradation signal: a view whose review fails stays pending (and
// captured — no data loss), and the next pass with a healthy endpoint
// reviews exactly the deferred view, clearing Pending.
func TestPassPendingCountsFailedReviews(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	dataDir := filepath.Join(base, "data")
	reviewsDir := filepath.Join(base, "reviews")
	shots := filepath.Join(base, "shots")

	view := filepath.Join(shots, "Home--dark--desktop.png")
	writePNGFile(t, view, 0x40)

	endpoint := newFlippingEndpoint(false)
	server := httptest.NewServer(endpoint)
	t.Cleanup(server.Close)

	config := fakeModelConfig(server.URL)

	// Retry off: the point is the failed-review accounting, not the backoff.
	disabled := RetryConfig{Enabled: false}
	config.Retry = &disabled

	ctx := context.Background()

	reviewer, err := NewReviewerFromConfig(ctx, config)
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

	failed, err := pipeline.Pass(ctx, projects)
	if err == nil {
		t.Fatal("pass against a 500ing endpoint must fail")
	}

	if failed.Captured != 1 || failed.Reviewed != 0 || failed.Pending != 1 {
		t.Fatalf("failed pass = %+v, want captured 1, reviewed 0, pending 1", failed)
	}

	endpoint.setHealthy(true)

	recovered, err := pipeline.Pass(ctx, projects)
	if err != nil {
		t.Fatalf("recovery pass: %v", err)
	}

	if recovered.Captured != 0 || recovered.Reviewed != 1 || recovered.Pending != 0 || recovered.Skipped != 0 {
		t.Fatalf("recovered pass = %+v, want captured 0, reviewed 1 (the deferred view), pending 0", recovered)
	}
}

// instantPassRunner finishes immediately, isolating the daemon's timing
// accounting from pipeline work.
type instantPassRunner struct{}

func (instantPassRunner) Pass(_ context.Context, _ map[string][]string) (PassResult, error) {
	return PassResult{}, nil
}

// TestRunPassWarnsWhenOverrun pins the dropped-tick warning: a pass measured
// longer than the interval must surface "overran the interval" in the logs.
func TestRunPassWarnsWhenOverrun(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&logs, nil))

	daemon, err := NewDaemon(
		instantPassRunner{},
		map[string][]string{"app": {"/nonexistent/*.png"}},
		time.Nanosecond,
		logger,
	)
	if err != nil {
		t.Fatalf("new daemon: %v", err)
	}

	daemon.runPass(context.Background())

	if !strings.Contains(logs.String(), "overran the interval") {
		t.Fatalf("want overran-interval warning in logs, got:\n%s", logs.String())
	}
}
