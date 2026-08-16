package reviewed

import (
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()

	store, err := OpenStore(filepath.Join(t.TempDir(), "events.bbolt"), slog.Default())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})

	return store
}

func testViewKey() ViewKey {
	return ViewKey{Page: "Settings", Theme: "dark", Viewport: "desktop"}
}

func TestStoreRecordAndLoadView(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := t.Context()
	viewKey := testViewKey()

	captured := Captured{
		SourcePath: "/shots/Settings--dark--desktop.png",
		BlobPath:   "/data/images/aaa.png",
		SHA256:     "aaa",
		CapturedAt: testStamp,
	}

	if err := store.RecordCapture(ctx, "discordsync", viewKey, captured); err != nil {
		t.Fatalf("record capture: %v", err)
	}

	state, version, err := store.LoadView(ctx, "discordsync", viewKey)
	if err != nil {
		t.Fatalf("load view: %v", err)
	}

	if state.Captures != 1 || state.SHA256 != "aaa" || state.BlobPath != captured.BlobPath {
		t.Fatalf("state after capture = %+v", state)
	}

	if !state.NeedsReview() {
		t.Fatal("fresh capture should need review")
	}

	if version != 1 {
		t.Fatalf("version = %d, want 1", version)
	}

	reviewed := Reviewed{
		SHA256:     "aaa",
		Model:      "test-model",
		Markdown:   "## Summary\nok",
		Score:      7,
		ReviewedAt: testStamp,
	}

	if err := store.RecordReview(ctx, "discordsync", viewKey, reviewed); err != nil {
		t.Fatalf("record review: %v", err)
	}

	state, version, err = store.LoadView(ctx, "discordsync", viewKey)
	if err != nil {
		t.Fatalf("load view after review: %v", err)
	}

	if state.NeedsReview() {
		t.Fatal("reviewed capture should not need review")
	}

	if state.LastReview == nil || state.LastReview.Markdown != "## Summary\nok" {
		t.Fatalf("last review = %+v", state.LastReview)
	}

	if state.LastScore != 7 {
		t.Fatalf("last score = %d, want 7", state.LastScore)
	}

	if version != 2 {
		t.Fatalf("version = %d, want 2", version)
	}
}

func TestStoreNewCaptureInvalidatesReview(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := t.Context()
	viewKey := testViewKey()

	first := Captured{SourcePath: "/a.png", BlobPath: "/b1", SHA256: "one", CapturedAt: testStamp}
	if err := store.RecordCapture(ctx, "proj", viewKey, first); err != nil {
		t.Fatalf("record capture 1: %v", err)
	}

	review := Reviewed{SHA256: "one", Score: 5, ReviewedAt: testStamp}
	if err := store.RecordReview(ctx, "proj", viewKey, review); err != nil {
		t.Fatalf("record review: %v", err)
	}

	second := Captured{SourcePath: "/a.png", BlobPath: "/b2", SHA256: "two", CapturedAt: testStamp.Add(time.Hour)}
	if err := store.RecordCapture(ctx, "proj", viewKey, second); err != nil {
		t.Fatalf("record capture 2: %v", err)
	}

	state, _, err := store.LoadView(ctx, "proj", viewKey)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if !state.NeedsReview() {
		t.Fatal("changed capture should need review again")
	}

	if state.Captures != 2 {
		t.Fatalf("captures = %d, want 2", state.Captures)
	}

	if state.SHA256 != "two" {
		t.Fatalf("sha = %q, want two", state.SHA256)
	}
}

func TestStoreScoreTrend(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := t.Context()
	viewKey := testViewKey()

	capture := Captured{SourcePath: "/a.png", SHA256: "one", CapturedAt: testStamp}
	if err := store.RecordCapture(ctx, "proj", viewKey, capture); err != nil {
		t.Fatalf("record capture: %v", err)
	}

	review := Reviewed{SHA256: "one", Score: 6, ReviewedAt: testStamp}
	if err := store.RecordReview(ctx, "proj", viewKey, review); err != nil {
		t.Fatalf("record review 1: %v", err)
	}

	state, _, err := store.LoadView(ctx, "proj", viewKey)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if state.PrevScore != ScoreUnknown || state.LastScore != 6 {
		t.Fatalf("after first review: last=%d prev=%d, want 6/%d", state.LastScore, state.PrevScore, ScoreUnknown)
	}

	review.Score = 8
	if err := store.RecordReview(ctx, "proj", viewKey, review); err != nil {
		t.Fatalf("record review 2: %v", err)
	}

	state, _, err = store.LoadView(ctx, "proj", viewKey)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if state.LastScore != 8 || state.PrevScore != 6 {
		t.Fatalf("after second review: last=%d prev=%d, want 8/6", state.LastScore, state.PrevScore)
	}
}

func TestStoreComparison(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := t.Context()
	viewKey := testViewKey()

	compared := Compared{
		BeforeSHA256: "one",
		AfterSHA256:  "two",
		Model:        "test-model",
		Markdown:     "## What improved\n- spacing",
		ComparedAt:   testStamp,
	}

	if err := store.RecordComparison(ctx, "proj", viewKey, compared); err != nil {
		t.Fatalf("record comparison: %v", err)
	}

	state, _, err := store.LoadView(ctx, "proj", viewKey)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if state.Comparisons != 1 {
		t.Fatalf("comparisons = %d, want 1", state.Comparisons)
	}
}

func TestStoreViewEvents(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := t.Context()
	viewKey := testViewKey()

	if err := store.RecordCapture(ctx, "proj", viewKey, Captured{SHA256: "one", CapturedAt: testStamp}); err != nil {
		t.Fatalf("record capture: %v", err)
	}

	review := Reviewed{SHA256: "one", Score: 5, ReviewedAt: testStamp}
	if err := store.RecordReview(ctx, "proj", viewKey, review); err != nil {
		t.Fatalf("record review: %v", err)
	}

	events, err := store.ViewEvents(ctx, "proj", viewKey)
	if err != nil {
		t.Fatalf("view events: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}

	if events[0].Type() != EventViewCaptured || events[1].Type() != EventViewReviewed {
		t.Fatalf("event types = %s, %s", events[0].Type(), events[1].Type())
	}
}

func TestStoreAllEventsAcrossStreams(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := t.Context()

	first := ViewKey{Page: "A", Theme: "dark", Viewport: "desktop"}
	second := ViewKey{Page: "B", Theme: "dark", Viewport: "desktop"}

	if err := store.RecordCapture(ctx, "proj1", first, Captured{SHA256: "one", CapturedAt: testStamp}); err != nil {
		t.Fatalf("record 1: %v", err)
	}

	if err := store.RecordCapture(ctx, "proj2", second, Captured{SHA256: "two", CapturedAt: testStamp}); err != nil {
		t.Fatalf("record 2: %v", err)
	}

	events, err := store.AllEvents(ctx)
	if err != nil {
		t.Fatalf("all events: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("journal = %d events, want 2", len(events))
	}
}

func TestStoreUnknownViewIsEmpty(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)

	state, version, err := store.LoadView(t.Context(), "proj", testViewKey())
	if err != nil {
		t.Fatalf("load unknown view: %v", err)
	}

	if state.Captures != 0 || version != 0 {
		t.Fatalf("unknown view should be empty, got %+v v%d", state, version)
	}
}

func TestStorePersistsAcrossReopen(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "events.bbolt")
	viewKey := testViewKey()

	first, err := OpenStore(path, slog.Default())
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	captured := Captured{SHA256: "one", CapturedAt: testStamp}
	if err := first.RecordCapture(t.Context(), "proj", viewKey, captured); err != nil {
		t.Fatalf("record: %v", err)
	}

	if err := first.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	second, err := OpenStore(path, slog.Default())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}

	defer func() {
		_ = second.Close()
	}()

	state, _, err := second.LoadView(t.Context(), "proj", viewKey)
	if err != nil {
		t.Fatalf("load after reopen: %v", err)
	}

	if state.Captures != 1 {
		t.Fatalf("captures after reopen = %d, want 1", state.Captures)
	}
}

func TestApplyViewStateIgnoresUnknownEventType(t *testing.T) {
	t.Parallel()

	state := ViewState{Captures: 3}
	event := unknownTestEvent(t)

	applied, err := ApplyViewState(state, event)
	if err != nil {
		t.Fatalf("unknown event type should not error: %v", err)
	}

	if applied.Captures != 3 {
		t.Fatalf("state should be untouched, got %+v", applied)
	}
}

func unknownTestEvent(t *testing.T) event.Event {
	t.Helper()

	streamID, err := ViewStreamID("proj", testViewKey())
	if err != nil {
		t.Fatalf("stream id: %v", err)
	}

	streamType, err := id.ParseStreamType(StreamTypeView)
	if err != nil {
		t.Fatalf("stream type: %v", err)
	}

	events, err := event.Single("view.unknown", streamID, streamType, 1, map[string]any{})
	if err != nil {
		t.Fatalf("single: %v", err)
	}

	return events[0]
}
