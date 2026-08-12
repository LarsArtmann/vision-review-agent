package catalog

import (
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/stretchr/testify/require"
)

func TestNewReturnsEmbeddedProviders(t *testing.T) {
	t.Parallel()

	svc := New()
	providers := svc.Providers()

	require.NotEmpty(t, providers, "embedded catalog must contain providers")
	require.Greater(t, len(providers), 10, "expected 40+ providers from embedded catalog")
}

func TestFindProviderKnown(t *testing.T) {
	t.Parallel()

	svc := New()

	tests := []struct {
		name string
		id   string
		want catwalk.Type
	}{
		{name: "openai", id: "openai", want: catwalk.TypeOpenAI},
		{name: "anthropic", id: "anthropic", want: catwalk.TypeAnthropic},
		{name: "gemini", id: "gemini", want: catwalk.TypeGoogle},
		{name: "openrouter", id: "openrouter", want: catwalk.TypeOpenRouter},
		{name: "xai", id: "xai", want: catwalk.TypeOpenAICompat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, ok := svc.FindProvider(tt.id)
			require.True(t, ok, "provider %q must be found", tt.id)
			require.Equal(t, tt.want, p.Type)
			require.NotEmpty(t, p.Models, "provider %q must have models", tt.id)
		})
	}
}

func TestFindProviderCaseInsensitive(t *testing.T) {
	t.Parallel()

	svc := New()

	tests := []string{"OpenAI", "ANTHROPIC", "Gemini", "OPENROUTER"}

	for _, id := range tests {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			p, ok := svc.FindProvider(id)
			require.True(t, ok, "provider lookup must be case-insensitive for %q", id)
			require.NotEmpty(t, p.Name)
		})
	}
}

func TestFindProviderUnknownReturnsFalse(t *testing.T) {
	t.Parallel()

	svc := New()

	p, ok := svc.FindProvider("nonexistent-provider")
	require.False(t, ok)
	require.Equal(t, catwalk.Provider{}, p)
}

func TestFindModelKnown(t *testing.T) {
	t.Parallel()

	svc := New()

	p, m, ok := svc.FindModel("gpt-4o")
	require.True(t, ok)
	require.Equal(t, "openai", string(p.ID))
	require.Equal(t, "gpt-4o", m.ID)
	require.True(t, m.SupportsImages, "gpt-4o must support images")
}

func TestFindModelCaseInsensitive(t *testing.T) {
	t.Parallel()

	svc := New()

	_, m, ok := svc.FindModel("GPT-4O")
	require.True(t, ok)
	require.Equal(t, "gpt-4o", m.ID)
}

func TestFindModelUnknownReturnsFalse(t *testing.T) {
	t.Parallel()

	svc := New()

	_, _, ok := svc.FindModel("gpt-999-not-real")
	require.False(t, ok)
}

func TestVisionModelsReturnsOnlyImageCapable(t *testing.T) {
	t.Parallel()

	svc := New()

	entries := svc.VisionModels()
	require.NotEmpty(t, entries, "catalog must have vision-capable models")

	for _, e := range entries {
		require.True(t, e.Model.SupportsImages,
			"VisionModels must only return models with SupportsImages=true, got %s/%s",
			e.Provider.ID, e.Model.ID)
	}
}

func TestVisionModelsIncludesKnownModels(t *testing.T) {
	t.Parallel()

	svc := New()

	entries := svc.VisionModels()

	known := map[string]bool{
		"gpt-4o":           false,
		"claude-mythos-5":  false,
		"gemini-3.6-flash": false,
	}

	for _, e := range entries {
		if _, ok := known[e.Model.ID]; ok {
			known[e.Model.ID] = true
		}
	}

	for model, found := range known {
		require.True(t, found, "vision models must include %q", model)
	}
}

func TestNewWithProvidersForTesting(t *testing.T) {
	t.Parallel()

	custom := []catwalk.Provider{
		{
			Name: "Test Provider",
			ID:   "test",
			Type: catwalk.TypeOpenAI,
			Models: []catwalk.Model{
				{ID: "test-model", Name: "Test", SupportsImages: true},
			},
		},
	}

	svc := NewWithProviders(custom)

	p, ok := svc.FindProvider("test")
	require.True(t, ok)
	require.Equal(t, "Test Provider", p.Name)

	entries := svc.VisionModels()
	require.Len(t, entries, 1)
	require.Equal(t, "test-model", entries[0].Model.ID)
}
