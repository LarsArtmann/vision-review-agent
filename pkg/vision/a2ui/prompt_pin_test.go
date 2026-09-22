package a2ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// The basic catalog is a moving upstream artifact. These tests pin the
// package to it: catalogSignatures must cover exactly the component kinds of
// the pinned official catalog (testdata/official/catalog.json), so upstream
// additions/removals fail loudly here instead of silently making the prompt
// wrong. When this test fails after a testdata refresh, update
// catalogSignatures and the builders.

// basicCatalogKinds extracts the component kind names from the pinned
// official basic catalog.
func basicCatalogKinds(t *testing.T) map[string]bool {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "official", "catalog.json"))
	require.NoError(t, err, "read pinned basic catalog")

	var catalog struct {
		Components map[string]json.RawMessage `json:"components"`
	}
	require.NoError(t, json.Unmarshal(data, &catalog))
	require.NotEmpty(t, catalog.Components)

	kinds := make(map[string]bool, len(catalog.Components))
	for kind := range catalog.Components {
		kinds[kind] = true
	}

	return kinds
}

// signatureKindRegexp matches the leading "Kind:" of a signature line.
// Multi-word guidance lines ("Heading shortcuts: ...") do not match.
var signatureKindRegexp = regexp.MustCompile(`^(\w+):`)

// TestCatalogSignaturesCoverBasicCatalog asserts the prompt's component
// signatures list exactly the pinned basic catalog's kinds: every kind is
// documented, and no signature invents a kind the catalog does not define.
func TestCatalogSignaturesCoverBasicCatalog(t *testing.T) {
	t.Parallel()

	expected := basicCatalogKinds(t)
	covered := make(map[string]bool, len(expected))

	for _, signature := range catalogSignatures() {
		match := signatureKindRegexp.FindStringSubmatch(signature)
		if match == nil {
			// Guidance lines are allowed; they must not look like kinds.
			require.NotRegexp(t, `^[A-Za-z]+:`, signature,
				"guidance line must not start with a single-word prefix: %q", signature)

			continue
		}

		kind := match[1]
		require.Truef(t, expected[kind],
			"signature %q is not a component of the pinned basic catalog", signature)
		require.Falsef(t, covered[kind], "duplicate signature for kind %q", kind)
		covered[kind] = true
	}

	for kind := range expected {
		require.Truef(t, covered[kind],
			"basic catalog kind %q has no signature in catalogSignatures", kind)
	}
}

// TestBasicCatalogKindCount documents the expected catalog size so drift is
// visible even in short test output.
func TestBasicCatalogKindCount(t *testing.T) {
	t.Parallel()

	require.Len(t, basicCatalogKinds(t), 18)
}

// TestCatalogSignaturesGuidanceLine pins the one intentional non-signature
// line (Text variants replace heading components).
func TestCatalogSignaturesGuidanceLine(t *testing.T) {
	t.Parallel()

	require.Contains(
		t,
		catalogSignatures(),
		"Heading shortcuts: use Text with variant h1..h5 instead of inventing heading components",
	)
}
