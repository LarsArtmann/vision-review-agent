package reviewed

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// backupTestEvents seeds a store with the canonical capture→review history
// and returns the view it used.
func seedBackupStore(t *testing.T, store *Store) ViewKey {
	t.Helper()

	ctx := t.Context()
	viewKey := ViewKey{Page: "Home", Theme: "dark", Viewport: "desktop"}

	captured := Captured{
		SourcePath: "/shots/Home--dark--desktop.png",
		BlobPath:   "images/bb1.png",
		SHA256:     "sha-before",
		CapturedAt: testStamp,
	}
	if err := store.RecordCapture(ctx, "proj", viewKey, captured); err != nil {
		t.Fatalf("record capture: %v", err)
	}

	reviewedPayload := Reviewed{
		SHA256:     "sha-before",
		Model:      "m",
		Markdown:   "## Review\n\n**Score: 7/10**",
		Score:      7,
		ReviewedAt: testStamp.Add(time.Minute),
	}
	if err := store.RecordReview(ctx, "proj", viewKey, reviewedPayload); err != nil {
		t.Fatalf("record review: %v", err)
	}

	return viewKey
}

// requireStoreStateEqual folds the same view in both stores and fails on any
// difference in state or event history.
func requireStoreStateEqual(t *testing.T, want, got *Store, project string, viewKey ViewKey) {
	t.Helper()

	wantState, wantVersion, err := want.LoadView(t.Context(), project, viewKey)
	if err != nil {
		t.Fatalf("load from source: %v", err)
	}

	gotState, gotVersion, err := got.LoadView(t.Context(), project, viewKey)
	if err != nil {
		t.Fatalf("load from restored: %v", err)
	}

	if wantVersion != gotVersion {
		t.Fatalf("version = %d, want %d", gotVersion, wantVersion)
	}

	requireViewStateEqual(t, gotState, wantState)

	wantEvents, err := want.ViewEvents(t.Context(), project, viewKey)
	if err != nil {
		t.Fatalf("source events: %v", err)
	}

	gotEvents, err := got.ViewEvents(t.Context(), project, viewKey)
	if err != nil {
		t.Fatalf("restored events: %v", err)
	}

	if len(wantEvents) != len(gotEvents) {
		t.Fatalf("restored journal has %d events, want %d", len(gotEvents), len(wantEvents))
	}

	for i := range wantEvents {
		if wantEvents[i].Type() != gotEvents[i].Type() || wantEvents[i].Version() != gotEvents[i].Version() {
			t.Fatalf("event %d differs: got %s v%d, want %s v%d",
				i, gotEvents[i].Type(), gotEvents[i].Version().Int(),
				wantEvents[i].Type(), wantEvents[i].Version().Int())
		}

		if !bytes.Equal(event.PayloadReadOnly(wantEvents[i]), event.PayloadReadOnly(gotEvents[i])) {
			t.Fatalf("event %d payload differs", i)
		}
	}
}

// TestStoreBackupRestoresIntoWorkingStore proves a Backup snapshot is a
// fully functional event store: it folds to the same state, replays the same
// history, and accepts NEW events with the right next version.
func TestStoreBackupRestoresIntoWorkingStore(t *testing.T) {
	t.Parallel()

	source := openTestStore(t)
	viewKey := seedBackupStore(t, source)

	var snapshot bytes.Buffer
	if err := source.Backup(t.Context(), &snapshot); err != nil {
		t.Fatalf("backup: %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "backup.db")
	if err := os.WriteFile(backupPath, snapshot.Bytes(), 0o600); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	restored, err := OpenStore(backupPath, quietLogger())
	if err != nil {
		t.Fatalf("open restored store: %v", err)
	}

	requireStoreStateEqual(t, source, restored, "proj", viewKey)

	_, version, err := restored.LoadView(t.Context(), "proj", viewKey)
	if err != nil {
		t.Fatalf("load restored: %v", err)
	}

	if err := restored.Close(); err != nil {
		t.Fatalf("close restored store: %v", err)
	}

	if version != 2 {
		t.Fatalf("restored version = %d, want 2", version)
	}

	newCapture := Captured{
		SourcePath: "/shots/Home--dark--desktop.png",
		BlobPath:   "images/bb2.png",
		SHA256:     "sha-after",
		CapturedAt: testStamp.Add(time.Hour),
	}

	if err := openAndRecord(t, backupPath, newCapture, viewKey); err != nil {
		t.Fatalf("append to restored store: %v", err)
	}
}

func openAndRecord(t *testing.T, path string, captured Captured, viewKey ViewKey) error {
	t.Helper()

	store, err := OpenStore(path, quietLogger())
	if err != nil {
		t.Fatalf("reopen restored store: %v", err)
	}

	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("close restored store: %v", err)
		}
	}()

	if err := store.RecordCapture(t.Context(), "proj", viewKey, captured); err != nil {
		return err
	}

	state, version, err := store.LoadView(t.Context(), "proj", viewKey)
	if err != nil {
		return err
	}

	if state.SHA256 != captured.SHA256 || version != 3 {
		t.Fatalf("after append: sha=%s version=%d, want %s v3", state.SHA256, version, captured.SHA256)
	}

	return nil
}

// TestStoreBackupWhileSourceOpen proves the raw read-only snapshot path
// reports a held journal quickly (bounded lock timeout) instead of hanging
// or silently snapshotting nothing: bbolt allows exactly one read-write
// handle and the daemon holds it, so the daemon must be stopped first.
func TestStoreBackupWhileSourceOpen(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	journal := filepath.Join(dir, "events.db")

	source, err := OpenStore(journal, quietLogger())
	if err != nil {
		t.Fatalf("open source: %v", err)
	}

	t.Cleanup(func() {
		if err := source.Close(); err != nil {
			t.Fatalf("close source: %v", err)
		}
	})

	seedBackupStore(t, source)

	var snapshot bytes.Buffer

	err = BackupJournalFile(journal, &snapshot, 100*time.Millisecond)
	if err == nil {
		t.Fatal("backup against a held journal should fail, not hang")
	}

	if snapshot.Len() != 0 {
		t.Fatal("failed backup must not write partial output")
	}
}

// TestBackupJournalFileSnapshotsRestorableStore proves the command-path
// snapshot (read-only raw bbolt handle) restores into a working store.
func TestBackupJournalFileSnapshotsRestorableStore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	journal := filepath.Join(dir, "events.db")

	source, err := OpenStore(journal, quietLogger())
	if err != nil {
		t.Fatalf("open source: %v", err)
	}

	viewKey := seedBackupStore(t, source)

	if err := source.Close(); err != nil {
		t.Fatalf("close source before backup: %v", err)
	}

	var snapshot bytes.Buffer
	if err := BackupJournalFile(journal, &snapshot, time.Second); err != nil {
		t.Fatalf("backup: %v", err)
	}

	backupPath := filepath.Join(dir, "backup.db")
	if err := os.WriteFile(backupPath, snapshot.Bytes(), 0o600); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	restored, err := OpenStore(backupPath, quietLogger())
	if err != nil {
		t.Fatalf("open restored store: %v", err)
	}

	t.Cleanup(func() {
		if err := restored.Close(); err != nil {
			t.Fatalf("close restored store: %v", err)
		}
	})

	if _, version, err := restored.LoadView(t.Context(), "proj", viewKey); err != nil || version != 2 {
		t.Fatalf("restored fold: version=%d err=%v, want version 2, nil", version, err)
	}
}

// TestJournalPath pins the journal file name inside the data dir.
func TestJournalPath(t *testing.T) {
	t.Parallel()

	if got, want := JournalPath("/data/visionreviewd"), "/data/visionreviewd/events.db"; got != want {
		t.Fatalf("JournalPath = %q, want %q", got, want)
	}
}

// TestBackupEventOrderStable guards the backup parity contract at the
// payload level: every decoded event type survives the roundtrip in order.
func TestBackupEventOrderStable(t *testing.T) {
	t.Parallel()

	source := openTestStore(t)
	viewKey := seedBackupStore(t, source)

	var snapshot bytes.Buffer
	if err := source.Backup(t.Context(), &snapshot); err != nil {
		t.Fatalf("backup: %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "backup.db")
	if err := os.WriteFile(backupPath, snapshot.Bytes(), 0o600); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	restored, err := OpenStore(backupPath, quietLogger())
	if err != nil {
		t.Fatalf("open restored: %v", err)
	}

	t.Cleanup(func() {
		if err := restored.Close(); err != nil {
			t.Fatalf("close restored: %v", err)
		}
	})

	wantTypes := []string{EventViewCaptured, EventViewReviewed}

	restoredEvents, err := restored.ViewEvents(t.Context(), "proj", viewKey)
	if err != nil {
		t.Fatalf("restored events: %v", err)
	}

	gotTypes := make([]string, 0, len(restoredEvents))
	for _, evt := range restoredEvents {
		gotTypes = append(gotTypes, string(evt.Type()))
	}

	if !slices.Equal(gotTypes, wantTypes) {
		t.Fatalf("restored event types = %v, want %v", gotTypes, wantTypes)
	}
}
