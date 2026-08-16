package reviewed

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Review file permissions: readable by the owner and group; the reviews are
// the product, meant to be consumed by humans and agents.
const (
	reviewsDirPermission  = 0o750
	reviewsFilePermission = 0o640
)

// comparisonTimeFormat stamps comparison filenames.
const comparisonTimeFormat = "2006-01-02_1504"

// Writer writes review markdown files under a reviews directory. All writes
// are atomic (temp file + rename) so readers never see partial files.
type Writer struct {
	reviewsDir string
}

// NewWriter returns a Writer rooted at reviewsDir.
func NewWriter(reviewsDir string) *Writer {
	return &Writer{reviewsDir: reviewsDir}
}

// ViewReviewPath returns the path of a view's review file.
func (w *Writer) ViewReviewPath(project string, viewKey ViewKey) string {
	return filepath.Join(w.reviewsDir, sanitizeName(project), "views", viewKey.String()+".md")
}

// ComparisonPath returns the path of a comparison file, stamped with the
// comparison time.
func (w *Writer) ComparisonPath(project string, viewKey ViewKey, at time.Time) string {
	name := fmt.Sprintf("%s_%s.md", at.UTC().Format(comparisonTimeFormat), viewKey)

	return filepath.Join(w.reviewsDir, sanitizeName(project), "comparisons", name)
}

// IndexPath returns the path of a project's INDEX.md.
func (w *Writer) IndexPath(project string) string {
	return filepath.Join(w.reviewsDir, sanitizeName(project), "INDEX.md")
}

// WriteViewReview atomically writes a view's review file.
func (w *Writer) WriteViewReview(project string, viewKey ViewKey, content string) error {
	return w.write(w.ViewReviewPath(project, viewKey), content)
}

// WriteComparison atomically writes a comparison file.
func (w *Writer) WriteComparison(project string, viewKey ViewKey, at time.Time, content string) error {
	return w.write(w.ComparisonPath(project, viewKey, at), content)
}

// WriteIndex atomically writes a project's INDEX.md.
func (w *Writer) WriteIndex(project string, content string) error {
	return w.write(w.IndexPath(project), content)
}

func (w *Writer) write(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), reviewsDirPermission); err != nil {
		return fmt.Errorf("create review dir %s: %w", filepath.Dir(path), err)
	}

	if err := writeFileAtomic(path, []byte(content), reviewsFilePermission); err != nil {
		return err
	}

	return nil
}

// writeFileAtomic writes data to a temporary file next to path and renames
// it into place.
func writeFileAtomic(path string, data []byte, permission os.FileMode) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".review-*")
	if err != nil {
		return fmt.Errorf("create temp next to %s: %w", path, err)
	}

	tempName := temp.Name()

	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempName)

		return fmt.Errorf("write %s: %w", tempName, err)
	}

	if err := temp.Close(); err != nil {
		_ = os.Remove(tempName)

		return fmt.Errorf("close %s: %w", tempName, err)
	}

	if err := os.Chmod(tempName, permission); err != nil {
		_ = os.Remove(tempName)

		return fmt.Errorf("chmod %s: %w", tempName, err)
	}

	if err := os.Rename(tempName, path); err != nil {
		_ = os.Remove(tempName)

		return fmt.Errorf("rename into %s: %w", path, err)
	}

	return nil
}

// sanitizeName makes a project name safe as a single path segment: lowercase
// alphanumerics, dashes, and underscores, with consecutive replacements
// squeezed into one dash.
func sanitizeName(project string) string {
	sanitized := make([]rune, 0, len(project))

	for _, r := range project {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			sanitized = append(sanitized, r)
		case r >= 'A' && r <= 'Z':
			sanitized = append(sanitized, r+'a'-'A')
		default:
			if len(sanitized) > 0 && sanitized[len(sanitized)-1] != '-' {
				sanitized = append(sanitized, '-')
			}
		}
	}

	sanitized = trimDashes(sanitized)

	if len(sanitized) == 0 {
		return "project"
	}

	return string(sanitized)
}

func trimDashes(runes []rune) []rune {
	start := 0
	for start < len(runes) && runes[start] == '-' {
		start++
	}

	end := len(runes)
	for end > start && runes[end-1] == '-' {
		end--
	}

	return runes[start:end]
}
