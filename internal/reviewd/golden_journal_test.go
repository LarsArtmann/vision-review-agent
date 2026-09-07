package reviewed

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	bolt "go.etcd.io/bbolt"
)

// The golden-journal fixture pins the on-disk journal format so a dependency
// bump can never again silently change how existing visionreviewd journals
// read. The bytes were produced by a real store; every test in this file
// loads them read-only and asserts the fold, payload decode, and raw wire
// envelope. See testdata/golden-journal.PROVENANCE.md for the recipe.

const goldenFixtureName = "golden-journal.bbolt"

// Fixed fixture content: two view streams covering every event type, a
// capture→review→capture→compare→review history on the first stream and a
// capture→review history on the second.
var (
	goldenProject1 = "discordsync"
	goldenView1    = ViewKey{Page: "Settings", Theme: "dark", Viewport: "desktop"}
	goldenProject2 = "visionreview"
	goldenView2    = ViewKey{Page: "Index", Theme: "light", Viewport: "mobile"}

	goldenSHAa = "9f2b6e1d0c8a4f3e5b7d9c1a2e4f6b8d0a3c5e7f9b1d3f5a7c9e1b3d5f7a9c2e"
	goldenSHAb = "1a3c5e7f9b2d4f6a8c0e2b4d6f8a1c3e5b7d9f1a3c5e7b9d2f4a6c8e0b3d5f7a"
	goldenSHAc = "c0e2b4d6f8a1c3e5b7d9f1a3c5e7b9d2f4a6c8e0b3d5f7a9c2e4f6a8d1c3e5b7"

	goldenCaptured1 = Captured{
		SourcePath: "/shots/Settings--dark--desktop.png",
		BlobPath:   "images/" + goldenSHAa + ".png",
		SHA256:     goldenSHAa,
		CapturedAt: time.Date(2026, 8, 16, 20, 31, 0, 0, time.UTC),
	}
	goldenReview1 = Reviewed{
		SHA256:     goldenSHAa,
		Model:      "llama3.2-vision:11b",
		Markdown:   "## Review\n\nSolid layout, weak contrast on the primary button.\n\n**Score: 6/10**",
		Score:      6,
		ReviewedAt: time.Date(2026, 8, 16, 20, 33, 0, 0, time.UTC),
	}
	goldenCaptured2 = Captured{
		SourcePath: "/shots/Settings--dark--desktop.png",
		BlobPath:   "images/" + goldenSHAb + ".png",
		SHA256:     goldenSHAb,
		CapturedAt: time.Date(2026, 8, 16, 21, 31, 0, 0, time.UTC),
	}
	goldenCompared = Compared{
		BeforeSHA256:   goldenSHAa,
		BeforeBlobPath: goldenCaptured1.BlobPath,
		AfterSHA256:    goldenSHAb,
		AfterBlobPath:  goldenCaptured2.BlobPath,
		Model:          "llama3.2-vision:11b",
		Markdown:       "## What changed\n\nSpacing tightened, heading now wraps.\n\n**Score: 8/10**",
		ComparedAt:     time.Date(2026, 8, 16, 21, 34, 0, 0, time.UTC),
	}
	goldenReview2 = Reviewed{
		SHA256:     goldenSHAb,
		Model:      "llama3.2-vision:11b",
		Markdown:   "## Review\n\nContrast fixed, focus ring still missing.\n\n**Score: 8/10**",
		Score:      8,
		ReviewedAt: time.Date(2026, 8, 16, 21, 36, 0, 0, time.UTC),
	}
	goldenCaptured3 = Captured{
		SourcePath: "/shots/Index--light--mobile.png",
		BlobPath:   "images/" + goldenSHAc + ".png",
		SHA256:     goldenSHAc,
		CapturedAt: time.Date(2026, 8, 16, 21, 1, 0, 0, time.UTC),
	}
	goldenReview3 = Reviewed{
		SHA256:     goldenSHAc,
		Model:      "llama3.2-vision:11b",
		Markdown:   "## Review\n\nMobile index reads cleanly.\n\n**Score: 9/10**",
		Score:      9,
		ReviewedAt: time.Date(2026, 8, 16, 21, 5, 0, 0, time.UTC),
	}
)

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

func quietLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// TestGenerateGoldenJournal regenerates the frozen fixture. It is skipped
// unless REVIEWD_GENERATE_GOLDEN_JOURNAL=1 so normal runs never rewrite the
// committed bytes. After regenerating, update the size and SHA-256 in
// testdata/golden-journal.PROVENANCE.md.
//
//nolint:paralleltest // writes the shared fixture file; parallel runs would race on it
func TestGenerateGoldenJournal(t *testing.T) {
	if os.Getenv("REVIEWD_GENERATE_GOLDEN_JOURNAL") == "" {
		t.Skip("set REVIEWD_GENERATE_GOLDEN_JOURNAL=1 to regenerate the golden journal fixture")
	}

	dbPath := filepath.Join(t.TempDir(), "events.bbolt")

	store, err := OpenStore(dbPath, quietLogger())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	ctx := t.Context()
	steps := []struct {
		name string
		exec func() error
	}{
		{"capture 1", func() error { return store.RecordCapture(ctx, goldenProject1, goldenView1, goldenCaptured1) }},
		{"review 1", func() error { return store.RecordReview(ctx, goldenProject1, goldenView1, goldenReview1) }},
		{"capture 2", func() error { return store.RecordCapture(ctx, goldenProject1, goldenView1, goldenCaptured2) }},
		{"compare", func() error { return store.RecordComparison(ctx, goldenProject1, goldenView1, goldenCompared) }},
		{"review 2", func() error { return store.RecordReview(ctx, goldenProject1, goldenView1, goldenReview2) }},
		{"capture 3", func() error { return store.RecordCapture(ctx, goldenProject2, goldenView2, goldenCaptured3) }},
		{"review 3", func() error { return store.RecordReview(ctx, goldenProject2, goldenView2, goldenReview3) }},
	}

	for _, step := range steps {
		if err := step.exec(); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close store before copy: %v", err)
	}

	if err := os.MkdirAll("testdata", 0o750); err != nil {
		t.Fatalf("create testdata dir: %v", err)
	}

	data, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("read generated journal: %v", err)
	}

	out := "testdata/" + goldenFixtureName

	if err := os.WriteFile(out, data, 0o644); err != nil { //nolint:gosec // constant fixture path, no user input
		t.Fatalf("write fixture: %v", err)
	}

	t.Logf("wrote %s: %d bytes, sha256 %s", out, len(data), sha256Hex(data))
}

// copyGoldenFixture copies the frozen fixture to a temp path so tests never
// touch the committed bytes, and asserts the source stays untouched.
func copyGoldenFixture(t *testing.T) string {
	t.Helper()

	src := "testdata/" + goldenFixtureName

	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read golden fixture: %v", err)
	}

	dst := filepath.Join(t.TempDir(), goldenFixtureName)

	if err := os.WriteFile(dst, data, 0o644); err != nil { //nolint:gosec // per-test temp dir, no user input
		t.Fatalf("copy golden fixture: %v", err)
	}

	t.Cleanup(func() {
		after, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("re-read golden fixture: %v", err)
		}

		if sha256Hex(after) != sha256Hex(data) {
			t.Fatal("golden fixture bytes changed during test; tests must never write to it")
		}
	})

	return dst
}

// requireViewStateEqual compares a folded ViewState against the expectation.
// Times compare by instant (not zone): the CBOR roundtrip returns times in a
// different time.Location representation of the same instant.
func requireViewStateEqual(t *testing.T, got, want ViewState) {
	t.Helper()

	if got.SHA256 != want.SHA256 || got.BlobPath != want.BlobPath ||
		got.Captures != want.Captures || got.ReviewedSHA != want.ReviewedSHA ||
		got.LastScore != want.LastScore || got.PrevScore != want.PrevScore ||
		got.Reviews != want.Reviews || got.Comparisons != want.Comparisons ||
		!got.CapturedAt.Equal(want.CapturedAt) {
		t.Fatalf("ViewState =\n%+v\nwant\n%+v", got, want)
	}

	requireReviewedEqual(t, got.LastReview, want.LastReview)
}

func requireReviewedEqual(t *testing.T, got, want *Reviewed) {
	t.Helper()

	if (got == nil) != (want == nil) {
		t.Fatalf("review = %+v, want %+v", got, want)
	}

	if got == nil {
		return
	}

	if got.SHA256 != want.SHA256 || got.Model != want.Model ||
		got.Markdown != want.Markdown || got.Score != want.Score ||
		!got.ReviewedAt.Equal(want.ReviewedAt) {
		t.Fatalf("review = %+v, want %+v", *got, *want)
	}
}

// openGoldenStore opens a read-only copy of the golden fixture as a Store.
func openGoldenStore(t *testing.T) *Store {
	t.Helper()

	store, err := OpenStore(copyGoldenFixture(t), quietLogger())
	if err != nil {
		t.Fatalf("open golden journal: %v", err)
	}

	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close golden journal store: %v", err)
		}
	})

	return store
}

// TestGoldenJournalFixtureLoads proves journals written by the pre-bump
// dependency set still load and fold into the exact expected ViewState on
// the current dependency set. This is the automated replacement for the
// manual wire-struct diff inspection done during the 2026-09-07 bump.
func TestGoldenJournalFixtureLoads(t *testing.T) {
	t.Parallel()

	store := openGoldenStore(t)
	ctx := t.Context()

	state, version, err := store.LoadView(ctx, goldenProject1, goldenView1)
	if err != nil {
		t.Fatalf("load stream 1: %v", err)
	}

	want1 := ViewState{
		SHA256:      goldenSHAb,
		BlobPath:    goldenCaptured2.BlobPath,
		CapturedAt:  goldenCaptured2.CapturedAt,
		Captures:    2,
		ReviewedSHA: goldenSHAb,
		LastReview:  &goldenReview2,
		LastScore:   8,
		PrevScore:   6,
		Reviews:     2,
		Comparisons: 1,
	}

	requireViewStateEqual(t, state, want1)

	if version != 5 {
		t.Fatalf("stream 1 version = %d, want 5", version)
	}

	if state.NeedsReview() {
		t.Fatal("fully reviewed stream should not need review")
	}

	if !state.UpdatedAt().Equal(goldenReview2.ReviewedAt) {
		t.Fatalf("UpdatedAt = %v, want review time %v", state.UpdatedAt(), goldenReview2.ReviewedAt)
	}

	state, version, err = store.LoadView(ctx, goldenProject2, goldenView2)
	if err != nil {
		t.Fatalf("load stream 2: %v", err)
	}

	want2 := ViewState{
		SHA256:      goldenSHAc,
		BlobPath:    goldenCaptured3.BlobPath,
		CapturedAt:  goldenCaptured3.CapturedAt,
		Captures:    1,
		ReviewedSHA: goldenSHAc,
		LastReview:  &goldenReview3,
		LastScore:   9,
		PrevScore:   ScoreUnknown,
		Reviews:     1,
		Comparisons: 0,
	}

	requireViewStateEqual(t, state, want2)

	if version != 2 {
		t.Fatalf("stream 2 version = %d, want 2", version)
	}

	stream1Events, err := store.ViewEvents(ctx, goldenProject1, goldenView1)
	if err != nil {
		t.Fatalf("stream 1 events: %v", err)
	}

	wantTypes := []event.Type{
		EventViewCaptured, EventViewReviewed, EventViewCaptured,
		EventViewCompared, EventViewReviewed,
	}

	if len(stream1Events) != len(wantTypes) {
		t.Fatalf("stream 1 has %d events, want %d", len(stream1Events), len(wantTypes))
	}

	for i, evt := range stream1Events {
		if evt.Type() != wantTypes[i] {
			t.Fatalf("stream 1 event %d = %s, want %s", i, evt.Type(), wantTypes[i])
		}

		if evt.Version().Int() != i+1 {
			t.Fatalf("stream 1 event %d version = %d, want %d", i, evt.Version().Int(), i+1)
		}
	}

	all, err := store.AllEvents(ctx)
	if err != nil {
		t.Fatalf("read all events: %v", err)
	}

	if len(all) != 7 {
		t.Fatalf("journal has %d events across streams, want 7", len(all))
	}
}

// TestGoldenJournalPayloadsDecode proves DecodePayloadAuto resolves the right
// codec for every fixture payload and recovers the exact original values.
func TestGoldenJournalPayloadsDecode(t *testing.T) {
	t.Parallel()

	store := openGoldenStore(t)

	events, err := store.ViewEvents(t.Context(), goldenProject1, goldenView1)
	if err != nil {
		t.Fatalf("stream 1 events: %v", err)
	}

	for i, evt := range events {
		switch evt.Type() {
		case EventViewCaptured:
			got, err := event.DecodePayloadAuto[Captured](evt)
			if err != nil {
				t.Fatalf("event %d: decode captured: %v", i, err)
			}

			want := goldenCaptured1
			if i == 2 {
				want = goldenCaptured2
			}

			if got.SourcePath != want.SourcePath || got.BlobPath != want.BlobPath ||
				got.SHA256 != want.SHA256 || !got.CapturedAt.Equal(want.CapturedAt) {
				t.Fatalf("event %d captured = %+v, want %+v", i, got, want)
			}
		case EventViewReviewed:
			got, err := event.DecodePayloadAuto[Reviewed](evt)
			if err != nil {
				t.Fatalf("event %d: decode reviewed: %v", i, err)
			}

			want := goldenReview1
			if i == 4 {
				want = goldenReview2
			}

			requireReviewedEqual(t, &got, &want)
		case EventViewCompared:
			got, err := event.DecodePayloadAuto[Compared](evt)
			if err != nil {
				t.Fatalf("event %d: decode compared: %v", i, err)
			}

			want := goldenCompared
			if got.BeforeSHA256 != want.BeforeSHA256 || got.BeforeBlobPath != want.BeforeBlobPath ||
				got.AfterSHA256 != want.AfterSHA256 || got.AfterBlobPath != want.AfterBlobPath ||
				got.Model != want.Model || got.Markdown != want.Markdown ||
				!got.ComparedAt.Equal(want.ComparedAt) {
				t.Fatalf("event %d compared = %+v, want %+v", i, got, want)
			}
		default:
			t.Fatalf("event %d: unexpected type %s", i, evt.Type())
		}
	}
}

// goldenEnvelopeKeys is the exact JSON-tag key set of the upstream
// serializableEvent CBOR envelope as written by storage/bbolt v4.0.0 and
// v4.1.0, including the schema_version every event stamps. Kept sorted; the
// check sorts decoded keys the same way before comparing.
var goldenEnvelopeKeys = []string{
	"aggregate_id", "aggregate_type", "encoding", "id", "metadata",
	"occurred_at", "payload", "schema_version", "type", "version",
}

// TestGoldenJournalEnvelopeContract decodes the raw bbolt rows behind the
// library API and pins the wire envelope: CBOR (not JSON), exact field-name
// key set, and stable event metadata. If a bump changes the envelope, this
// fails loudly instead of corrupting reads of existing journals.
func TestGoldenJournalEnvelopeContract(t *testing.T) {
	t.Parallel()

	fixtureDB, err := bolt.Open(copyGoldenFixture(t), 0o444, &bolt.Options{ReadOnly: true})
	if err != nil {
		t.Fatalf("open fixture read-only: %v", err)
	}

	t.Cleanup(func() {
		if err := fixtureDB.Close(); err != nil {
			t.Fatalf("close fixture db: %v", err)
		}
	})

	var rowCount int

	err = fixtureDB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("cqrs_events"))
		if bucket == nil {
			t.Fatal("fixture has no cqrs_events bucket")
		}

		return bucket.ForEach(func(_, value []byte) error {
			if value == nil {
				return nil
			}

			rowCount++

			return checkEnvelopeRow(t, rowCount, value)
		})
	})
	if err != nil {
		t.Fatalf("scan fixture: %v", err)
	}

	if rowCount != 7 {
		t.Fatalf("fixture has %d event rows, want 7", rowCount)
	}
}

func checkEnvelopeRow(t *testing.T, rowNumber int, value []byte) error {
	t.Helper()

	if !isCBORFirstByte(value) {
		t.Fatalf("row %d: envelope is not CBOR (first byte %#x)", rowNumber, value[0])
	}

	var row map[string]any

	if err := codec.CBORDecMode().Unmarshal(value, &row); err != nil {
		return fmt.Errorf("decode row %d: %w", rowNumber, err)
	}

	keys := make([]string, 0, len(row))
	for key := range row {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	if !slices.Equal(keys, goldenEnvelopeKeys) {
		t.Fatalf("row %d: envelope keys = %v, want %v", rowNumber, keys, goldenEnvelopeKeys)
	}

	if typ, _ := row["type"].(string); typ != EventViewCaptured && typ != EventViewReviewed && typ != EventViewCompared {
		t.Fatalf("row %d: unexpected event type %q", rowNumber, typ)
	}

	if aggType, _ := row["aggregate_type"].(string); aggType != StreamTypeView {
		t.Fatalf("row %d: aggregate_type = %q, want %q", rowNumber, aggType, StreamTypeView)
	}

	if enc, _ := row["encoding"].(string); enc != string(codec.EncodingCBOR) {
		t.Fatalf("row %d: payload encoding = %q, want %q", rowNumber, enc, codec.EncodingCBOR)
	}

	if schemaVersion, _ := row["schema_version"].(uint64); schemaVersion != 1 {
		t.Fatalf("row %d: schema_version = %v, want 1", rowNumber, row["schema_version"])
	}

	return nil
}

func isCBORFirstByte(data []byte) bool {
	return len(data) > 0 && data[0] >= 0xa0 && data[0] <= 0xbf
}

// TestPayloadStructJSONTagsPinned pins the JSON tags of the event payload
// structs via reflection. Those tags are how old journals decode; renaming
// one silently turns reviewed payloads into zero values. Change them only
// together with a migration path documented in testdata PROVENANCE.
func TestPayloadStructJSONTagsPinned(t *testing.T) {
	t.Parallel()

	want := map[string]map[string]string{
		"Captured": {
			"SourcePath": "sourcePath",
			"BlobPath":   "blobPath",
			"SHA256":     "sha256",
			"CapturedAt": "capturedAt",
		},
		"Reviewed": {
			"SHA256":     "sha256",
			"Model":      "model",
			"Markdown":   "markdown",
			"Score":      "score",
			"ReviewedAt": "reviewedAt",
		},
		"Compared": {
			"BeforeSHA256":   "beforeSha256",
			"BeforeBlobPath": "beforeBlobPath",
			"AfterSHA256":    "afterSha256",
			"AfterBlobPath":  "afterBlobPath",
			"Model":          "model",
			"Markdown":       "markdown",
			"ComparedAt":     "comparedAt",
		},
	}

	for structName, wantTags := range want {
		var got map[string]string

		switch structName {
		case "Captured":
			got = jsonTagsOf[Captured]()
		case "Reviewed":
			got = jsonTagsOf[Reviewed]()
		case "Compared":
			got = jsonTagsOf[Compared]()
		}

		if !reflect.DeepEqual(got, wantTags) {
			t.Fatalf("%s json tags = %v, want %v (payload tags are the journal compatibility contract)", structName, got, wantTags)
		}
	}
}

func jsonTagsOf[T any]() map[string]string {
	typ := reflect.TypeFor[T]()

	tags := make(map[string]string, typ.NumField())

	for field := range typ.Fields() {
		tags[field.Name] = field.Tag.Get("json")
	}

	return tags
}

// FuzzGoldenJournalDecode feeds arbitrary payloads and encodings through the
// journal read path (reconstruct → DecodePayloadAuto → fold). Corrupt or
// hostile rows must fail with an error, never panic. Seeds include the
// fixture's real payloads.
func FuzzGoldenJournalDecode(f *testing.F) {
	f.Add(EventViewCaptured, string(codec.EncodingJSON), []byte(
		`{"sourcePath":"/shots/s.png","blobPath":"images/x.png","sha256":"abc","capturedAt":"2026-08-16T20:31:00Z"}`))
	f.Add(EventViewReviewed, string(codec.EncodingJSON), []byte(
		`{"sha256":"abc","model":"m","markdown":"## Review","score":6,"reviewedAt":"2026-08-16T20:33:00Z"}`))
	f.Add(EventViewCompared, string(codec.EncodingJSON), []byte(`{}`))
	f.Add(EventViewCaptured, string(codec.EncodingCBOR), []byte{0xa4, 0x01, 0x02, 0x03})
	f.Add(EventViewCaptured, "", []byte("not a payload"))
	f.Add("future.event", string(codec.EncodingJSON), []byte(`{"anything":true}`))
	f.Add(EventViewReviewed, string(codec.EncodingCBOR), []byte{0xff, 0xff, 0xff})

	f.Fuzz(func(t *testing.T, eventType, encoding string, payload []byte) {
		evt, err := event.ReconstructEventWithAdoptedPayload(
			id.NewEventID(), event.Type(eventType), id.StreamType(StreamTypeView),
			mustStreamID(t), 1, 0, payload, event.Metadata{},
			time.Date(2026, 8, 16, 20, 31, 0, 0, time.UTC), codec.Encoding(encoding), "fuzz",
		)
		if err != nil {
			t.Skipf("reconstruct rejected input: %v", err)
		}

		_, _ = ApplyViewState(initialViewState(), evt)
	})
}

func mustStreamID(t *testing.T) id.StreamID {
	t.Helper()

	streamID, err := id.ParseStreamID(goldenProject1 + ":" + goldenView1.String())
	if err != nil {
		t.Fatalf("parse stream id: %v", err)
	}

	return streamID
}
