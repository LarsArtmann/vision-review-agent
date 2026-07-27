package main

import (
	"testing"
	"time"

	"github.com/larsartmann/vision-review-agent/pkg/vision"
	"github.com/stretchr/testify/require"
)

func TestAdviceForKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind vision.ErrorKind
		want string
	}{
		{name: "rate limited", kind: vision.KindRateLimited, want: "rate-limiting"},
		{name: "timeout", kind: vision.KindTimeout, want: "timed out"},
		{name: "server error", kind: vision.KindServerError, want: "experiencing issues"},
		{name: "service unavailable", kind: vision.KindServiceUnavailable, want: "temporarily unavailable"},
		{name: "not implemented", kind: vision.KindNotImplemented, want: "does not implement"},
		{name: "network", kind: vision.KindNetwork, want: "internet connection"},
		{name: "authentication", kind: vision.KindAuthentication, want: "API key"},
		{name: "not found", kind: vision.KindNotFound, want: "model was not found"},
		{name: "bad request", kind: vision.KindBadRequest, want: "rejected the request"},
		{name: "content filter", kind: vision.KindContentFilter, want: "content policy"},
		{name: "context too large", kind: vision.KindContextTooLarge, want: "context window"},
		{name: "cancelled", kind: vision.KindCancelled, want: "cancelled"},
		{name: "structured parse", kind: vision.KindStructuredParse, want: "structured output"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			advice := adviceForKind(tt.kind)
			require.NotEmpty(t, advice, "every known kind should have advice")
			require.Contains(t, advice, tt.want)
		})
	}
}

func TestAdviceForKindUnknownReturnsEmpty(t *testing.T) {
	t.Parallel()

	require.Empty(t, adviceForKind(vision.ErrorKind("something_unknown")))
}

func TestBuildConfig(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		cfg := &config{temperature: 0.3, maxTokens: 0}
		built := buildConfig(nil, cfg)
		require.InDelta(t, 0.3, built.Temperature, 0.001)
		require.Equal(t, int64(0), built.MaxOutputTokens)
		require.Empty(t, built.SystemPrompt)
		require.Equal(t, time.Duration(0), built.RequestTimeout)
	})

	t.Run("with system prompt", func(t *testing.T) {
		t.Parallel()

		cfg := &config{systemPrompt: "You are a UI expert."}
		built := buildConfig(nil, cfg)
		require.Equal(t, "You are a UI expert.", built.SystemPrompt)
	})

	t.Run("with timeout", func(t *testing.T) {
		t.Parallel()

		cfg := &config{timeout: 30}
		built := buildConfig(nil, cfg)
		require.Equal(t, 30*time.Second, built.RequestTimeout)
	})

	t.Run("zero timeout does not set RequestTimeout", func(t *testing.T) {
		t.Parallel()

		cfg := &config{timeout: 0}
		built := buildConfig(nil, cfg)
		require.Equal(t, time.Duration(0), built.RequestTimeout)
	})

	t.Run("with max tokens", func(t *testing.T) {
		t.Parallel()

		cfg := &config{maxTokens: 4096}
		built := buildConfig(nil, cfg)
		require.Equal(t, int64(4096), built.MaxOutputTokens)
	})
}

func TestParseTimeout(t *testing.T) {
	t.Parallel()

	require.Equal(t, 10*time.Second, parseTimeout(10))
	require.Equal(t, time.Duration(0), parseTimeout(0))
	require.Equal(t, 90*time.Second, parseTimeout(90))
}

func TestCreateProviderUnknown(t *testing.T) {
	t.Parallel()

	_, err := createProvider("unknown-provider")
	require.Error(t, err)
	require.ErrorIs(t, err, errUnknownProvider)
}

func TestCreateProviderOpenAIMissingKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")

	_, err := createProvider("openai")
	require.Error(t, err)
	require.ErrorIs(t, err, errEnvVarNotSet)
}
