package reviewed

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBlobStoreStoreNewBlob(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	src := filepath.Join(dir, "shot.png")
	if err := os.WriteFile(src, []byte("fake png bytes"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	store := NewBlobStore(dir)

	sha, blobPath, err := store.Store(src)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	if len(sha) != 64 {
		t.Fatalf("sha length = %d, want 64", len(sha))
	}

	wantPath := filepath.Join(dir, "images", sha+".png")
	if blobPath != wantPath {
		t.Fatalf("blobPath = %q, want %q", blobPath, wantPath)
	}

	got, err := os.ReadFile(blobPath)
	if err != nil {
		t.Fatalf("read blob: %v", err)
	}

	if !bytes.Equal(got, []byte("fake png bytes")) {
		t.Fatalf("blob content = %q, want %q", got, "fake png bytes")
	}
}

func TestBlobStoreStoreSkipsExisting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	src := filepath.Join(dir, "shot.png")
	if err := os.WriteFile(src, []byte("stable content"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	store := NewBlobStore(dir)

	firstSha, firstPath, err := store.Store(src)
	if err != nil {
		t.Fatalf("first Store: %v", err)
	}

	// Corrupt the source after the first store: a skipped copy must not
	// overwrite the archived blob.
	if err := os.WriteFile(src, []byte("changed content"), 0o644); err != nil {
		t.Fatalf("rewrite src: %v", err)
	}

	// Store a file with the ORIGINAL content again by pointing at a copy.
	original := filepath.Join(dir, "original.png")
	if err := os.WriteFile(original, []byte("stable content"), 0o644); err != nil {
		t.Fatalf("write original: %v", err)
	}

	secondSha, secondPath, err := store.Store(original)
	if err != nil {
		t.Fatalf("second Store: %v", err)
	}

	if secondSha != firstSha || secondPath != firstPath {
		t.Fatalf("second store diverged: %q/%q, want %q/%q", secondSha, secondPath, firstSha, firstPath)
	}

	got, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("read blob: %v", err)
	}

	if !bytes.Equal(got, []byte("stable content")) {
		t.Fatalf("blob content = %q, want %q", got, "stable content")
	}
}

func TestBlobStoreDifferentContentDifferentBlob(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewBlobStore(dir)

	first := filepath.Join(dir, "a.png")
	second := filepath.Join(dir, "b.png")

	if err := os.WriteFile(first, []byte("one"), 0o644); err != nil {
		t.Fatalf("write first: %v", err)
	}

	if err := os.WriteFile(second, []byte("two"), 0o644); err != nil {
		t.Fatalf("write second: %v", err)
	}

	firstSha, _, err := store.Store(first)
	if err != nil {
		t.Fatalf("store first: %v", err)
	}

	secondSha, _, err := store.Store(second)
	if err != nil {
		t.Fatalf("store second: %v", err)
	}

	if firstSha == secondSha {
		t.Fatal("different content produced the same hash")
	}
}

func TestBlobStoreDefaultExtension(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewBlobStore(dir)

	src := filepath.Join(dir, "noext")
	if err := os.WriteFile(src, []byte("data"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	sha, blobPath, err := store.Store(src)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	wantPath := filepath.Join(dir, "images", sha+".png")
	if blobPath != wantPath {
		t.Fatalf("blobPath = %q, want %q", blobPath, wantPath)
	}
}

func TestBlobStoreMissingSource(t *testing.T) {
	t.Parallel()

	store := NewBlobStore(t.TempDir())

	if _, _, err := store.Store(filepath.Join(t.TempDir(), "missing.png")); err == nil {
		t.Fatal("Store on missing source should error")
	}
}

// TestBlobStoreCheckWritable proves the pass-start disk probe: a healthy
// (or creatable) blob dir passes, an unwritable location fails with the
// offending path named.
func TestBlobStoreCheckWritable(t *testing.T) {
	t.Parallel()

	t.Run("healthy dir passes and re-creates a wiped dataDir", func(t *testing.T) {
		t.Parallel()

		blobs := NewBlobStore(filepath.Join(t.TempDir(), "data"))

		if err := blobs.CheckWritable(); err != nil {
			t.Fatalf("fresh dataDir should pass: %v", err)
		}

		if err := os.Remove(blobs.Dir()); err != nil {
			t.Fatalf("remove blob dir: %v", err)
		}

		if err := blobs.CheckWritable(); err != nil {
			t.Fatalf("wiped blob dir should heal: %v", err)
		}
	})

	t.Run("unwritable location fails naming the path", func(t *testing.T) {
		t.Parallel()

		if os.Geteuid() == 0 {
			t.Skip("root ignores permission bits")
		}

		base := t.TempDir()
		readOnly := filepath.Join(base, "ro")

		if err := os.Mkdir(readOnly, 0o500); err != nil {
			t.Fatalf("mkdir ro: %v", err)
		}

		blobs := &BlobStore{dir: filepath.Join(readOnly, "images")}

		err := blobs.CheckWritable()
		if err == nil || !strings.Contains(err.Error(), readOnly) {
			t.Fatalf("err = %v, want failure naming %s", err, readOnly)
		}
	})
}
