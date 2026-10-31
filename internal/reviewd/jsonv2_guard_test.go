package reviewed

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// jsonv2ImportPattern matches the quoted wire-import form of the json/v2
// experiment paths. The v2 regime swaps encoding/json in place; importing the
// experiment paths directly needs a go.mod replace and a different low-level
// API, so the v1 path is the only supported form (see AGENTS.md, "Dual json
// v1+v2 support").
var jsonv2ImportPattern = regexp.MustCompile(`"encoding/json/(v2|jsontext)(/[a-z0-9]+)?"`)

// findJSONv2Imports walks dir and returns every .go file (testdata excluded)
// whose bytes contain a quoted json/v2 or jsontext import path.
func findJSONv2Imports(dir string) ([]string, error) {
	var offenders []string

	walkErr := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			if entry.Name() == "testdata" {
				return filepath.SkipDir
			}

			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		if jsonv2ImportPattern.Match(data) {
			offenders = append(offenders, path)
		}

		return nil
	})

	return offenders, walkErr
}

// TestNoJSONv2Imports guards the dual json v1+v2 contract inside the daemon
// package. The auto-commit daemon pushes directly to master and bypasses CI,
// so a package-local guard that runs on every go test is the only detection
// that cannot be routed around (seventh recurrence, 2026-10-11).
func TestNoJSONv2Imports(t *testing.T) {
	t.Parallel()

	offenders, err := findJSONv2Imports(".")
	if err != nil {
		t.Fatalf("walk package dir: %v", err)
	}

	for _, path := range offenders {
		t.Errorf("%s: forbidden json/v2 experiment import; use plain encoding/json (see AGENTS.md)", path)
	}
}

// TestFindJSONv2ImportsDetectsOffender is the positive control proving the
// guard can actually fail: a scratch directory seeded with a v2 import must
// be flagged.
func TestFindJSONv2ImportsDetectsOffender(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	clean := filepath.Join(dir, "clean.go")
	dirty := filepath.Join(dir, "dirty.go")
	nested := filepath.Join(dir, "testdata", "ignored.go")

	const cleanSource = "package scratch\n\nimport \"encoding/json\"\n"
	const dirtySource = "package scratch\n\nimport \"encoding/json/v2\"\n"

	if err := os.WriteFile(clean, []byte(cleanSource), 0o600); err != nil {
		t.Fatalf("write clean scratch file: %v", err)
	}

	if err := os.WriteFile(dirty, []byte(dirtySource), 0o600); err != nil {
		t.Fatalf("write dirty scratch file: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(nested), 0o700); err != nil {
		t.Fatalf("mkdir testdata scratch dir: %v", err)
	}

	if err := os.WriteFile(nested, []byte(dirtySource), 0o600); err != nil {
		t.Fatalf("write testdata scratch file: %v", err)
	}

	offenders, err := findJSONv2Imports(dir)
	if err != nil {
		t.Fatalf("walk scratch dir: %v", err)
	}

	if len(offenders) != 1 || offenders[0] != dirty {
		t.Fatalf("want exactly [%s], got %v (testdata must be skipped)", dirty, offenders)
	}
}
