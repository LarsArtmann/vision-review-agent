package cli

import (
	"context"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
)

// NewOpenAIModel creates an OpenAI language model, reading the API key from
// the OPENAI_API_KEY environment variable. It exits the process on any error.
//
// Call RequireArgc first so os.Args is valid if you also read a model from
// the command line.
func NewOpenAIModel(ctx context.Context, model string) fantasy.LanguageModel {
	apiKey := RequireEnvVar("OPENAI_API_KEY")

	provider, err := openai.New(openai.WithAPIKey(apiKey))
	ExitOnError(err, "Error creating provider")

	lm, err := provider.LanguageModel(ctx, model)
	ExitOnError(err, "Error getting model")

	return lm
}
