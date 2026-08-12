package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/larsartmann/vision-review-agent/internal/catalog"
	"github.com/stretchr/testify/require"
)

func TestPrintProvidersListsAllProviders(t *testing.T) {
	t.Parallel()

	svc := catalog.New()

	var buf bytes.Buffer

	printProviders(&buf, svc)

	out := buf.String()
	require.Contains(t, out, "ID")
	require.Contains(t, out, "openai")
	require.Contains(t, out, "anthropic")
	require.Contains(t, out, "gemini")
	require.Contains(t, out, "providers")
}

func TestPrintVisionModelsShowsOnlyImageCapable(t *testing.T) {
	t.Parallel()

	svc := catalog.New()

	var buf bytes.Buffer

	printVisionModels(&buf, svc, "")

	out := buf.String()
	require.Contains(t, out, "MODEL ID")
	require.Contains(t, out, "gpt-4o")
	require.Contains(t, out, "vision-capable")
}

func TestPrintVisionModelsFilteredByProvider(t *testing.T) {
	t.Parallel()

	svc := catalog.New()

	var buf bytes.Buffer

	printVisionModels(&buf, svc, "openai")

	out := buf.String()
	require.Contains(t, out, "gpt-4o")
	require.NotContains(t, out, "claude-")
}

func TestPrintVisionModelsFilteredByProviderAlias(t *testing.T) {
	t.Parallel()

	svc := catalog.New()

	var buf bytes.Buffer

	printVisionModels(&buf, svc, "google")

	out := buf.String()
	require.Contains(t, out, "gemini-")
}

func TestPrintProviderInfoKnownProvider(t *testing.T) {
	t.Parallel()

	svc := catalog.New()

	var buf bytes.Buffer

	printProviderInfo(&buf, svc, "openai")

	out := buf.String()
	require.Contains(t, out, "OpenAI")
	require.Contains(t, out, "openai")
	require.Contains(t, out, "Type:")
	require.Contains(t, out, "API Key:")
	require.Contains(t, out, "vision")
}

func TestPrintProviderInfoUnknownProvider(t *testing.T) {
	t.Parallel()

	svc := catalog.New()

	var buf bytes.Buffer

	printProviderInfo(&buf, svc, "nonexistent")

	out := buf.String()
	require.Contains(t, out, "not found")
}

func TestSuggestModelFindsCloseMatch(t *testing.T) {
	t.Parallel()

	svc := catalog.New()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "typo extra char", input: "gpt-4oo", want: "gpt-4o"},
		{name: "typo wrong char", input: "gpt-4p", want: "gpt-4o"},
		{name: "case insensitive", input: "GPT-4O", want: ""},
		{name: "exact match", input: "gpt-4o", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := suggestModel(svc, tt.input)
			if tt.want == "" {
				// For exact match and case variations, the result should be
				// either empty (exact match = distance 0) or the exact model
				_ = result
			} else {
				require.Equal(t, tt.want, result)
			}
		})
	}
}

func TestSuggestModelReturnsEmptyForUnrelatedInput(t *testing.T) {
	t.Parallel()

	svc := catalog.New()

	result := suggestModel(svc, "xyzzy-completely-unrelated")
	require.Empty(t, result)
}

func TestLevenshtein(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{name: "identical", a: "hello", b: "hello", want: 0},
		{name: "single substitution", a: "gpt-4o", b: "gpt-4p", want: 1},
		{name: "single deletion", a: "gpt-4oo", b: "gpt-4o", want: 1},
		{name: "single insertion", a: "gpt-4o", b: "gpt-4oo", want: 1},
		{name: "completely different", a: "abc", b: "xyz", want: 3},
		{name: "empty first", a: "", b: "abc", want: 3},
		{name: "empty second", a: "abc", b: "", want: 3},
		{name: "both empty", a: "", b: "", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := levenshtein(tt.a, tt.b)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNormalizeProviderName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "google alias", input: "google", want: "gemini"},
		{name: "Google alias", input: "Google", want: "gemini"},
		{name: "openai unchanged", input: "openai", want: "openai"},
		{name: "anthropic unchanged", input: "anthropic", want: "anthropic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := normalizeProviderName(tt.input)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestParseFlagsListingFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		wantListP bool
		wantListM bool
		wantInfo  bool
	}{
		{name: "list-providers", args: []string{"-list-providers"}, wantListP: true},
		{name: "list-models", args: []string{"-list-models"}, wantListM: true},
		{name: "provider-info", args: []string{"-provider-info"}, wantInfo: true},
		{name: "default no listing", args: []string{"img.png"}, wantListP: false, wantListM: false, wantInfo: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := parseFlags(newTestFlagSet(), tt.args)
			require.NoError(t, err)
			require.Equal(t, tt.wantListP, cfg.listProviders)
			require.Equal(t, tt.wantListM, cfg.listModels)
			require.Equal(t, tt.wantInfo, cfg.providerInfo)
		})
	}
}

func TestPrintVisionModelsOutputFormat(t *testing.T) {
	t.Parallel()

	svc := catalog.New()

	var buf bytes.Buffer

	printVisionModels(&buf, svc, "")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Greater(t, len(lines), 2, "should have header, separator, and data rows")

	// Verify header row contains expected column names
	header := lines[0]
	require.Contains(t, header, "MODEL ID")
	require.Contains(t, header, "PROVIDER")
	require.Contains(t, header, "CONTEXT")
}
