package catalog

import (
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/stretchr/testify/require"
)

func TestBuildProviderOpenAI(t *testing.T) {
	t.Parallel()

	provider, err := BuildProvider(catwalk.Provider{
		Type: catwalk.TypeOpenAI,
	}, "sk-test-key", "")

	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestBuildProviderAnthropic(t *testing.T) {
	t.Parallel()

	provider, err := BuildProvider(catwalk.Provider{
		Type: catwalk.TypeAnthropic,
	}, "sk-test-key", "")

	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestBuildProviderGoogleWithAPIKey(t *testing.T) {
	t.Parallel()

	provider, err := BuildProvider(catwalk.Provider{
		Type: catwalk.TypeGoogle,
	}, "test-gemini-key", "")

	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestBuildProviderGoogleWithoutAPIKey(t *testing.T) {
	t.Parallel()

	provider, err := BuildProvider(catwalk.Provider{
		Type: catwalk.TypeGoogle,
	}, "", "")

	require.NoError(t, err, "google should work without API key (ADC fallback)")
	require.NotNil(t, provider)
}

func TestBuildProviderOpenRouter(t *testing.T) {
	t.Parallel()

	provider, err := BuildProvider(catwalk.Provider{
		Type: catwalk.TypeOpenRouter,
	}, "sk-test-key", "")

	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestBuildProviderOpenAICompat(t *testing.T) {
	t.Parallel()

	provider, err := BuildProvider(catwalk.Provider{
		Type: catwalk.TypeOpenAICompat,
	}, "sk-test-key", "https://api.groq.com/openai/v1")

	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestBuildProviderOpenAICompatNoAPIKey(t *testing.T) {
	t.Parallel()

	provider, err := BuildProvider(catwalk.Provider{
		Type: catwalk.TypeOpenAICompat,
	}, "", "http://localhost:8080/v1")

	require.NoError(t, err, "openai-compat should work without API key (local servers)")
	require.NotNil(t, provider)
}

func TestBuildProviderUnsupportedType(t *testing.T) {
	t.Parallel()

	_, err := BuildProvider(catwalk.Provider{
		Type: catwalk.Type("unknown-type"),
		ID:   "test-provider",
	}, "key", "")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrUnsupportedType)
}

func TestBuildProviderWithBaseURL(t *testing.T) {
	t.Parallel()

	provider, err := BuildProvider(catwalk.Provider{
		Type: catwalk.TypeOpenAI,
	}, "sk-test-key", "https://custom.api.com/v1")

	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestResolveAPIKeyFromEnvVar(t *testing.T) {
	t.Setenv("TEST_CATWALK_KEY", "test-value-123")

	key, err := ResolveAPIKey(catwalk.Provider{
		APIKey: "$TEST_CATWALK_KEY",
	})
	require.NoError(t, err)
	require.Equal(t, "test-value-123", key)
}

func TestResolveAPIKeyMissingEnvVar(t *testing.T) {
	t.Setenv("MISSING_CATWALK_KEY", "")

	_, err := ResolveAPIKey(catwalk.Provider{
		APIKey: "$MISSING_CATWALK_KEY",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAPIKeyNotSet)
}

func TestResolveAPIKeyEmptyField(t *testing.T) {
	t.Parallel()

	key, err := ResolveAPIKey(catwalk.Provider{APIKey: ""})
	require.NoError(t, err)
	require.Empty(t, key)
}

func TestResolveAPIKeyLiteralKey(t *testing.T) {
	t.Parallel()

	key, err := ResolveAPIKey(catwalk.Provider{APIKey: "literal-key-no-dollar"})
	require.NoError(t, err)
	require.Equal(t, "literal-key-no-dollar", key)
}

func TestResolveBaseURLDirectURL(t *testing.T) {
	t.Parallel()

	url := ResolveBaseURL(catwalk.Provider{
		APIEndpoint: "https://api.x.ai/v1",
	})
	require.Equal(t, "https://api.x.ai/v1", url)
}

func TestResolveBaseURLFromEnvVar(t *testing.T) {
	t.Setenv("TEST_ENDPOINT", "https://custom.endpoint.com")

	url := ResolveBaseURL(catwalk.Provider{
		APIEndpoint: "$TEST_ENDPOINT",
	})
	require.Equal(t, "https://custom.endpoint.com", url)
}

func TestResolveBaseURLEmptyField(t *testing.T) {
	t.Parallel()

	url := ResolveBaseURL(catwalk.Provider{APIEndpoint: ""})
	require.Empty(t, url)
}

func TestResolveBaseURLUnsetEnvVar(t *testing.T) {
	t.Setenv("UNSET_ENDPOINT", "")

	url := ResolveBaseURL(catwalk.Provider{
		APIEndpoint: "$UNSET_ENDPOINT",
	})
	require.Empty(t, url)
}

func TestRequiresAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pType   catwalk.Type
		require bool
	}{
		{name: "openai", pType: catwalk.TypeOpenAI, require: true},
		{name: "anthropic", pType: catwalk.TypeAnthropic, require: true},
		{name: "openrouter", pType: catwalk.TypeOpenRouter, require: true},
		{name: "openai-compat", pType: catwalk.TypeOpenAICompat, require: true},
		{name: "google", pType: catwalk.TypeGoogle, require: false},
		{name: "vertex", pType: catwalk.TypeVertexAI, require: false},
		{name: "bedrock", pType: catwalk.TypeBedrock, require: false},
		{name: "azure", pType: catwalk.TypeAzure, require: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := RequiresAPIKey(catwalk.Provider{Type: tt.pType})
			require.Equal(t, tt.require, result)
		})
	}
}
