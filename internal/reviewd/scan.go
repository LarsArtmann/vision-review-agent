package reviewed

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Capture is one screenshot file the scanner found: which view it shows,
// where it lives, and its content hash.
type Capture struct {
	ViewKey ViewKey

	// Path is the file's location on disk at scan time.
	Path string

	// SHA256 is the hex hash of the file's bytes.
	SHA256 string

	// ModifiedAt is the file's modification time; it becomes the capture's
	// CapturedAt timestamp and breaks ties when several files show the same
	// view.
	ModifiedAt time.Time
}

// ScanProject expands a project's globs and returns one Capture per view,
// sorted by view key. When several files map to the same view key the newest
// modification time wins, so a view is always represented by its latest
// screenshot. Directories and non-regular files matched by a glob are
// ignored.
//
// Failures are isolated per glob and per file: one bad pattern or unreadable
// file no longer blinds the scan to everything after it. The returned error
// joins every failure (nil when none); the returned captures are everything
// that scanned successfully, so callers can process the good part while
// still surfacing the failures.
func ScanProject(globs []string) ([]Capture, error) {
	latest := make(map[ViewKey]Capture)

	var errs []error

	for _, glob := range globs {
		matches, err := filepath.Glob(glob)
		if err != nil {
			errs = append(errs, fmt.Errorf("scan glob %s: %w", glob, err))

			continue
		}

		for _, path := range matches {
			info, err := os.Stat(path)
			if err != nil {
				errs = append(errs, fmt.Errorf("scan %s: %w", path, err))

				continue
			}

			if !info.Mode().IsRegular() {
				continue
			}

			if !isScreenshotExtension(strings.ToLower(filepath.Ext(path))) {
				continue
			}

			capture, err := captureOf(path, info)
			if err != nil {
				errs = append(errs, err)

				continue
			}

			existing, seen := latest[capture.ViewKey]
			if !seen || capture.ModifiedAt.After(existing.ModifiedAt) {
				latest[capture.ViewKey] = capture
			}
		}
	}

	return sortedCaptures(latest), errors.Join(errs...)
}

// captureOf derives a view key from the file name and hashes the file.
func captureOf(path string, info os.FileInfo) (Capture, error) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	viewKey, err := ParseViewKey(name)
	if err != nil {
		return Capture{}, fmt.Errorf("scan %s: %w", path, err)
	}

	sha, err := SHA256File(path)
	if err != nil {
		return Capture{}, fmt.Errorf("scan %s: %w", path, err)
	}

	return Capture{
		ViewKey:    viewKey,
		Path:       path,
		SHA256:     sha,
		ModifiedAt: info.ModTime(),
	}, nil
}

// sortedCaptures returns the map's values ordered by view key.
func sortedCaptures(latest map[ViewKey]Capture) []Capture {
	captures := make([]Capture, 0, len(latest))

	for _, capture := range latest {
		captures = append(captures, capture)
	}

	sort.Slice(captures, func(i, j int) bool {
		return captures[i].ViewKey.String() < captures[j].ViewKey.String()
	})

	return captures
}
