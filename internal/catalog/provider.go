package catalog

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
	"charm.land/fantasy/providers/google"
	"charm.land/fantasy/providers/openai"
	"charm.land/fantasy/providers/openaicompat"
	"charm.land/fantasy/providers/openrouter"
)

// ErrAPIKeyNotSet is returned when the environment variable named in a
// catwalk Provider.APIKey field is not set.
var ErrAPIKeyNotSet = errors.New("environment variable not set")

// ErrUnsupportedType is returned when a catwalk Provider.Type does not map
// to a known fantasy provider constructor.
var ErrUnsupportedType = errors.New("unsupported provider type")

// BuildProvider constructs a fantasy.Provider from a catwalk.Provider metadata
// entry. It switches on the catwalk Type to call the appropriate fantasy
// constructor. The apiKey and baseURL parameters are pre-resolved values
// (typically from environment variables referenced in the catwalk data).
//
// For providers that support multiple auth mechanisms (e.g., Google with ADC),
// an empty apiKey is acceptable and the constructor falls back to its default
// auth behavior.
func BuildProvider(p catwalk.Provider, apiKey, baseURL string) (fantasy.Provider, error) {
	switch p.Type {
	case catwalk.TypeOpenAI:
		return buildOpenAIProvider(apiKey, baseURL)
	case catwalk.TypeAnthropic:
		return buildAnthropicProvider(apiKey, baseURL)
	case catwalk.TypeGoogle:
		return buildGoogleProvider(apiKey, baseURL)
	case catwalk.TypeOpenRouter:
		return buildOpenRouterProvider(apiKey)
	case catwalk.TypeOpenAICompat:
		return buildOpenAICompatProvider(apiKey, baseURL)
	case catwalk.TypeAzure, catwalk.TypeBedrock, catwalk.TypeVertexAI, catwalk.TypeVercel:
		return nil, fmt.Errorf("%w: %q (provider %s)", ErrUnsupportedType, p.Type, p.ID)
	default:
		return nil, fmt.Errorf("%w: %q (provider %s)", ErrUnsupportedType, p.Type, p.ID)
	}
}

func buildOpenAIProvider(apiKey, baseURL string) (fantasy.Provider, error) {
	opts := []openai.Option{openai.WithAPIKey(apiKey)}

	if baseURL != "" {
		opts = append(opts, openai.WithBaseURL(baseURL))
	}

	provider, err := openai.New(opts...)

	return wrapProvider("openai", provider, err)
}

func buildAnthropicProvider(apiKey, baseURL string) (fantasy.Provider, error) {
	opts := []anthropic.Option{anthropic.WithAPIKey(apiKey)}

	if baseURL != "" {
		opts = append(opts, anthropic.WithBaseURL(baseURL))
	}

	provider, err := anthropic.New(opts...)

	return wrapProvider("anthropic", provider, err)
}

func buildGoogleProvider(apiKey, baseURL string) (fantasy.Provider, error) {
	opts := []google.Option{}

	if apiKey != "" {
		opts = append(opts, google.WithGeminiAPIKey(apiKey))
	}

	if baseURL != "" {
		opts = append(opts, google.WithBaseURL(baseURL))
	}

	provider, err := google.New(opts...)

	return wrapProvider("google", provider, err)
}

func buildOpenRouterProvider(apiKey string) (fantasy.Provider, error) {
	provider, err := openrouter.New(openrouter.WithAPIKey(apiKey))

	return wrapProvider("openrouter", provider, err)
}

func buildOpenAICompatProvider(apiKey, baseURL string) (fantasy.Provider, error) {
	opts := []openaicompat.Option{}

	if baseURL != "" {
		opts = append(opts, openaicompat.WithBaseURL(baseURL))
	}

	if apiKey != "" {
		opts = append(opts, openaicompat.WithAPIKey(apiKey))
	}

	provider, err := openaicompat.New(opts...)

	return wrapProvider("openai-compat", provider, err)
}

func wrapProvider(name string, provider fantasy.Provider, err error) (fantasy.Provider, error) {
	if err != nil {
		return nil, fmt.Errorf("create %s provider: %w", name, err)
	}

	return provider, nil
}

// ResolveAPIKey extracts the API key from the environment variable named in
// the catwalk Provider.APIKey field. Catwalk stores env var names with a "$"
// prefix (e.g., "$OPENAI_API_KEY"). If the field is empty or the env var is
// unset, it returns "" with ErrAPIKeyNotSet.
//
// For providers that support auth fallbacks (Google ADC, local servers), the
// caller may treat the error as non-fatal and pass an empty apiKey to
// BuildProvider.
func ResolveAPIKey(p catwalk.Provider) (string, error) {
	if p.APIKey == "" {
		return "", nil
	}

	envVar, ok := strings.CutPrefix(p.APIKey, "$")
	if !ok {
		return p.APIKey, nil
	}

	apiKey := os.Getenv(envVar)
	if apiKey == "" {
		return "", fmt.Errorf("%s: %w", envVar, ErrAPIKeyNotSet)
	}

	return apiKey, nil
}

// ResolveBaseURL extracts the base URL from the catwalk Provider.APIEndpoint
// field. Catwalk stores either:
//   - An env var name with "$" prefix (e.g., "$ANTHROPIC_API_ENDPOINT") —
//     resolved from the environment; returns "" if unset (use provider default)
//   - A direct URL (e.g., "https://api.x.ai/v1") — returned as-is
//   - Empty — returns ""
func ResolveBaseURL(p catwalk.Provider) string {
	if p.APIEndpoint == "" {
		return ""
	}

	envVar, ok := strings.CutPrefix(p.APIEndpoint, "$")
	if !ok {
		return p.APIEndpoint
	}

	return os.Getenv(envVar)
}

// RequiresAPIKey returns true when the provider type needs an API key to
// function. Google and bedrock may use alternative auth (ADC, IAM roles).
func RequiresAPIKey(p catwalk.Provider) bool {
	switch p.Type {
	case
		catwalk.TypeGoogle,
		catwalk.TypeVertexAI,
		catwalk.TypeBedrock,
		catwalk.TypeAzure:
		return false
	case
		catwalk.TypeOpenAI,
		catwalk.TypeOpenAICompat,
		catwalk.TypeOpenRouter,
		catwalk.TypeAnthropic,
		catwalk.TypeVercel:
		return true
	default:
		return true
	}
}
