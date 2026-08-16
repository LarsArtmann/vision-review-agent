package reviewed

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// BlobStore archives screenshots content-addressed under
// <dataDir>/images/<sha256>.<ext>. Screenshot files are overwritten in place
// upstream, so copying every new capture here preserves the full visual
// history: event log plus blobs reconstruct any past state without the model.
type BlobStore struct {
	dir string
}

// NewBlobStore returns a BlobStore rooted at <dataDir>/images.
func NewBlobStore(dataDir string) *BlobStore {
	return &BlobStore{dir: filepath.Join(dataDir, "images")}
}

// Dir returns the directory blobs are stored in.
func (b *BlobStore) Dir() string {
	return b.dir
}

// Store copies src into the blob store under its content hash and returns the
// hex SHA-256 and the blob path. When the blob already exists the copy is
// skipped; identical content is stored exactly once.
func (b *BlobStore) Store(src string) (sha string, blobPath string, err error) {
	sha, err = SHA256File(src)
	if err != nil {
		return "", "", err
	}

	ext := strings.ToLower(filepath.Ext(src))
	if ext == "" {
		ext = ".png"
	}

	target := filepath.Join(b.dir, sha+ext)
	if _, statErr := os.Stat(target); statErr == nil {
		return sha, target, nil
	}

	if err := os.MkdirAll(b.dir, 0o755); err != nil {
		return "", "", fmt.Errorf("blob store mkdir %s: %w", b.dir, err)
	}

	if err := copyFileAtomic(src, target); err != nil {
		return "", "", err
	}

	return sha, target, nil
}

// copyFileAtomic copies src to a temporary file next to target and renames it
// into place, so readers never observe a partially written blob.
func copyFileAtomic(src, target string) error {
	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("blob store open %s: %w", src, err)
	}
	defer source.Close()

	temp, err := os.CreateTemp(filepath.Dir(target), ".blob-*")
	if err != nil {
		return fmt.Errorf("blob store temp next to %s: %w", target, err)
	}

	tempName := temp.Name()

	if _, err := io.Copy(temp, source); err != nil {
		_ = temp.Close()
		os.Remove(tempName)

		return fmt.Errorf("blob store copy %s: %w", src, err)
	}

	if err := temp.Close(); err != nil {
		os.Remove(tempName)

		return fmt.Errorf("blob store close %s: %w", tempName, err)
	}

	if err := os.Chmod(tempName, 0o644); err != nil {
		os.Remove(tempName)

		return fmt.Errorf("blob store chmod %s: %w", tempName, err)
	}

	if err := os.Rename(tempName, target); err != nil {
		os.Remove(tempName)

		return fmt.Errorf("blob store rename %s: %w", target, err)
	}

	return nil
}
