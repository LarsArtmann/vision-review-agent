package reviewed

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// SHA256File streams the file at path and returns its hex-encoded SHA-256
// digest. Streaming keeps memory flat for large screenshots.
func SHA256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}

	return hex.EncodeToString(digest.Sum(nil)), nil
}
