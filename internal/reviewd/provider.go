package reviewed

import (
	"context"
	"fmt"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openaicompat"
)

// LanguageModel builds the fantasy.LanguageModel that talks to the configured
// OpenAI-compatible endpoint (llama-server). The API key is optional:
// llama-server ignores it, other compatible servers may require it.
//
//nolint:ireturn // boundary function whose whole job is handing back the fantasy interface
func LanguageModel(ctx context.Context, config Config) (fantasy.LanguageModel, error) {
	opts := []openaicompat.Option{openaicompat.WithBaseURL(config.BaseURL)}

	if config.APIKey != "" {
		opts = append(opts, openaicompat.WithAPIKey(config.APIKey))
	}

	provider, err := openaicompat.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("create openai-compat provider for %s: %w", config.BaseURL, err)
	}

	languageModel, err := provider.LanguageModel(ctx, config.Model)
	if err != nil {
		return nil, fmt.Errorf("resolve model %s on %s: %w", config.Model, config.BaseURL, err)
	}

	return languageModel, nil
}
