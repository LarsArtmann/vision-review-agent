
import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestNoV5RemovedPairFormAPIs guards the go-cqrs-lite v5 migration. The
// decider repository's pair-form Load/Execute are deprecated in v4 and
// removed in v5; the ref-form (LoadRef/ExecuteRef with id.NewStreamRef) is
// the surviving API. The guard fails if a pair-form call sneaks back in,
// so the migration stays a lookup away instead of an archaeology project.
func TestNoV5RemovedPairFormAPIs(t *testing.T) {
	t.Parallel()

	pairForm := regexp.MustCompile(`repo\.(Load|Execute)\(`)

	walkErr := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
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

		data, readErr := os.ReadFile(path) //nolint:gosec // repo-relative path from WalkDir
		if readErr != nil {
			return readErr
		}

		if loc := pairForm.Find(data); loc != nil {
			t.Errorf("%s: pair-form API call removed in go-cqrs-lite v5 (use LoadRef/ExecuteRef): %s", path, data[loc[0]:min(loc[1], len(data))])
		}

		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk package dir: %v", walkErr)
	}
}
