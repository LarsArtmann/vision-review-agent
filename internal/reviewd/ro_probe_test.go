package reviewed

import (
	"path/filepath"
	"testing"
	"time"
)

func TestROProbe(t *testing.T) {
	dir := t.TempDir()
	journal := filepath.Join(dir, "events.db")

	source, err := OpenStore(journal, quietLogger())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = source.Close() })

	done := make(chan error, 1)
	go func() {
		reader, err := OpenStoreReadOnly(journal, quietLogger())
		if err != nil {
			done <- err
			return
		}
		done <- reader.Close()
	}()

	select {
	case err := <-done:
		t.Logf("OpenStoreReadOnly+Close returned: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("OpenStoreReadOnly hangs")
	}
}
