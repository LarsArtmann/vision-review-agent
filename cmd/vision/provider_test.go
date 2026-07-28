package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateProviderOpenRouterMissingKey(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")

	_, err := createProvider("openrouter")
	require.Error(t, err)
	require.ErrorIs(t, err, errEnvVarNotSet)
}

func TestCreateProviderAnthropicMissingKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")

	_, err := createProvider("anthropic")
	require.Error(t, err)
	require.ErrorIs(t, err, errEnvVarNotSet)
}

func TestCreateProviderOpenAIWithFakeKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-fake-test-key")

	provider, err := createProvider("openai")
	require.NoError(t, err, "provider construction must succeed even with a fake key")
	require.NotNil(t, provider)
}

func TestCreateProviderOpenRouterWithFakeKey(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "sk-fake-test-key")

	provider, err := createProvider("openrouter")
	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestCreateProviderAnthropicWithFakeKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-fake-test-key")

	provider, err := createProvider("anthropic")
	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestCreateProviderIsCaseInsensitive(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-fake-test-key")

	provider, err := createProvider("OpenAI")
	require.NoError(t, err, "provider name lookup must be case-insensitive")
	require.NotNil(t, provider)
}
