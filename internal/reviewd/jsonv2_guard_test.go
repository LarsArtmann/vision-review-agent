package reviewed

import (
	"errors"
	"fmt"
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
			// testdata holds fixtures, vendor/ holds dependencies whose
			// v2 imports are the documented transitive blocker — neither
			// is this repo's code.
			if entry.Name() == "testdata" || entry.Name() == "vendor" {
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

	const (
		cleanSource = "package scratch\n\nimport \"encoding/json\"\n"
		dirtySource = "package scratch\n\nimport \"encoding/json/v2\"\n"
	)

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

// errNoGoMod is the sentinel returned when findModuleRoot walks past the
// filesystem root without finding a go.mod (e.g. when tests run from a
// scratch directory outside any module).
var errNoGoMod = errors.New("no go.mod found above the working directory")

// findModuleRoot walks up from the working directory (the package dir during
// tests) until it finds go.mod, returning the module root.
func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working dir: %w", err)
	}

	return findModuleRootFrom(dir)
}

// TestNoJSONv2ImportsModuleWide extends the package-local guard to every Go
// file in the module. The migrator runs `fix ./...` from the module root, so
// cmd/, pkg/errors, internal/catalog, and examples are all in its blast
// radius even though the historical recurrences only landed in this package
// and pkg/vision (a2ui). This test catches those on the next
// `go test ./internal/reviewd/` instead of waiting for each victim package's
// own test run.
func TestNoJSONv2ImportsModuleWide(t *testing.T) {
	t.Parallel()

	root, err := findModuleRoot()
	if err != nil {
		t.Fatalf("locate module root: %v", err)
	}

	offenders, err := findJSONv2Imports(root)
	if err != nil {
		t.Fatalf("walk module %s: %v", root, err)
	}

	for _, path := range offenders {
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}

		t.Errorf("%s: forbidden json/v2 experiment import; use plain encoding/json (see AGENTS.md)", rel)
	}
}

// TestFindModuleRootControls pins findModuleRoot's two outcomes: a directory
// inside this module resolves to a go.mod-bearing root, and a scratch
// directory outside any module reports errNoGoMod.
func TestFindModuleRootControls(t *testing.T) {
	t.Parallel()

	root, err := findModuleRoot()
	if err != nil {
		t.Fatalf("findModuleRoot inside module: %v", err)
	}

	if _, statErr := os.Stat(filepath.Join(root, "go.mod")); statErr != nil {
		t.Fatalf("resolved root %s has no go.mod: %v", root, statErr)
	}

	if _, err := findModuleRootFrom(t.TempDir()); !errors.Is(err, errNoGoMod) {
		t.Fatalf("scratch dir: want errNoGoMod, got %v", err)
	}
}

// findModuleRootFrom is findModuleRoot with an explicit start directory so
// the failure path is testable without changing the process working
// directory.
func findModuleRootFrom(start string) (string, error) {
	dir := start

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errNoGoMod
		}

		dir = parent
	}
}
