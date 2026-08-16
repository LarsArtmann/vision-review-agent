package main

import (
	"flag"
	"io"
	"testing"
	"time"

	"github.com/larsartmann/vision-review-agent/internal/catalog"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
	"github.com/stretchr/testify/require"
)

// newTestFlagSet returns an isolated, quiet FlagSet for parseFlags tests.
func newTestFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("vision-test", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // silence usage output on parse errors

	return fs
}

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
		built := buildConfig(nil, cfg, nil)
		require.InEpsilon(t, 0.3, built.Temperature, 1e-9)
		require.Equal(t, int64(0), built.MaxOutputTokens)
		require.Empty(t, built.SystemPrompt)
		require.Equal(t, time.Duration(0), built.RequestTimeout)
	})

	t.Run("with system prompt", func(t *testing.T) {
		t.Parallel()

		cfg := &config{systemPrompt: "You are a UI expert."}
		built := buildConfig(nil, cfg, nil)
		require.Equal(t, "You are a UI expert.", built.SystemPrompt)
	})

	t.Run("with timeout", func(t *testing.T) {
		t.Parallel()

		cfg := &config{timeout: 30}
		built := buildConfig(nil, cfg, nil)
		require.Equal(t, 30*time.Second, built.RequestTimeout)
	})

	t.Run("zero timeout does not set RequestTimeout", func(t *testing.T) {
		t.Parallel()

		cfg := &config{timeout: 0}
		built := buildConfig(nil, cfg, nil)
		require.Equal(t, time.Duration(0), built.RequestTimeout)
	})

	t.Run("with max tokens", func(t *testing.T) {
		t.Parallel()

		cfg := &config{maxTokens: 4096}
		built := buildConfig(nil, cfg, nil)
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

	_, err := createProvider(catalog.New(), "unknown-provider")
	require.Error(t, err)
	require.ErrorIs(t, err, errUnknownProvider)
}

func TestCreateProviderOpenAIMissingKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")

	_, err := createProvider(catalog.New(), "openai")
	require.Error(t, err)
	require.ErrorIs(t, err, errEnvVarNotSet)
}

func TestParseFlagsDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := parseFlags(newTestFlagSet(), []string{"screenshot.png"})
	require.NoError(t, err)

	require.Equal(t, "openai", cfg.providerName)
	require.Equal(t, "gpt-4o", cfg.modelID)
	require.Equal(t, "Describe what you see in this image.", cfg.prompt)
	require.Empty(t, cfg.systemPrompt)
	require.False(t, cfg.stream)
	require.False(t, cfg.jsonOutput)
	require.False(t, cfg.structured)
	require.InEpsilon(t, defaultTemperature, cfg.temperature, 1e-9)
	require.Equal(t, int64(0), cfg.maxTokens)
	require.Equal(t, int64(0), cfg.timeout)
	require.Equal(t, []string{"screenshot.png"}, cfg.args)
	require.False(t, cfg.showVersion)
}

func TestParseFlagsAllFlags(t *testing.T) {
	t.Parallel()

	args := []string{
		"-provider", "openrouter",
		"-model", "anthropic/claude-3.5-sonnet",
		"-prompt", "Find UI bugs",
		"-system", "You are an accessibility expert",
		"-stream",
		"-temperature", "0.7",
		"-max-tokens", "2048",
		"-json",
		"-structured",
		"-timeout", "45",
		"a.png", "b.png",
	}

	cfg, err := parseFlags(newTestFlagSet(), args)
	require.NoError(t, err)

	require.Equal(t, "openrouter", cfg.providerName)
	require.Equal(t, "anthropic/claude-3.5-sonnet", cfg.modelID)
	require.Equal(t, "Find UI bugs", cfg.prompt)
	require.Equal(t, "You are an accessibility expert", cfg.systemPrompt)
	require.True(t, cfg.stream)
	require.True(t, cfg.jsonOutput)
	require.True(t, cfg.structured)
	require.InEpsilon(t, 0.7, cfg.temperature, 1e-9)
	require.Equal(t, int64(2048), cfg.maxTokens)
	require.Equal(t, int64(45), cfg.timeout)
	require.Equal(t, []string{"a.png", "b.png"}, cfg.args)
}

func TestParseFlagsVersion(t *testing.T) {
	t.Parallel()

	cfg, err := parseFlags(newTestFlagSet(), []string{"-version"})
	require.NoError(t, err)
	require.True(t, cfg.showVersion, "-version must set showVersion")
	require.Empty(t, cfg.args)
}

func TestParseFlagsNoPositionalArgs(t *testing.T) {
	t.Parallel()

	// parseFlags does not exit on missing positional args — it returns an empty
	// args slice. main() is responsible for printing usage and exiting.
	cfg, err := parseFlags(newTestFlagSet(), []string{})
	require.NoError(t, err)
	require.Empty(t, cfg.args, "no positional args must yield empty cfg.args")
}

func TestParseFlagsRejectsUnknownFlag(t *testing.T) {
	t.Parallel()

	_, err := parseFlags(newTestFlagSet(), []string{"-bogus", "img.png"})
	require.Error(t, err, "unknown flags must surface as a parse error")
}

func TestNormalizeProviderNameGoogleAlias(t *testing.T) {
	t.Parallel()

	require.Equal(t, "gemini", normalizeProviderName("google"))
	require.Equal(t, "gemini", normalizeProviderName("Google"))
	require.Equal(t, "gemini", normalizeProviderName("GOOGLE"))
	require.Equal(t, "openai", normalizeProviderName("openai"))
}

func TestFindModelInProviderWithGoogleAlias(t *testing.T) {
	t.Parallel()

	svc := catalog.New()

	// "google" must normalize to "gemini" for catalog lookup. Pick whatever
	// model the catalog currently lists instead of hardcoding an ID that
	// breaks whenever catwalk updates its data.
	normalized := normalizeProviderName("google")
	require.Equal(t, "gemini", normalized)

	provider, ok := svc.FindProvider(normalized)
	require.True(t, ok, "catalog must list the normalized %q provider", normalized)
	require.NotEmpty(t, provider.Models, "catalog %q provider must list models", normalized)

	_, found := svc.FindModelInProvider(normalized, provider.Models[0].ID)
	require.True(t, found, "FindModelInProvider must find models under normalized %q provider", normalized)
}
