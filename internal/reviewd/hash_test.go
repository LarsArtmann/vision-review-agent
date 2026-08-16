package reviewed

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestSHA256FileKnownVector(t *testing.T) {
	path := filepath.Join(t.TempDir(), "content.bin")
	if err := os.WriteFile(path, []byte("hello world"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	want := sha256.Sum256([]byte("hello world"))

	got, err := SHA256File(path)
	if err != nil {
		t.Fatalf("SHA256File: %v", err)
	}

	if wantHex := hex.EncodeToString(want[:]); got != wantHex {
		t.Fatalf("SHA256File = %q, want %q", got, wantHex)
	}
}

func TestSHA256FileEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.bin")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := SHA256File(path)
	if err != nil {
		t.Fatalf("SHA256File: %v", err)
	}

	// SHA-256 of the empty string.
	const emptySHA = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != emptySHA {
		t.Fatalf("SHA256File empty = %q, want %q", got, emptySHA)
	}
}

func TestSHA256FileMissing(t *testing.T) {
	if _, err := SHA256File(filepath.Join(t.TempDir(), "missing.png")); err == nil {
		t.Fatal("SHA256File on missing file should error")
	}
}
