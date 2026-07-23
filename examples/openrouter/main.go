// Example: OpenRouter vision analysis (supports many models)
//
// Usage:
//
//	export OPENROUTER_API_KEY=your-key
//	go run examples/openrouter/main.go screenshot.png
package main

import (
	"context"
	"fmt"
	"os"

	"charm.land/fantasy/providers/openrouter"
	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

func main() {
	cli.RequireArgc(2)

	apiKey := cli.RequireEnvVar("OPENROUTER_API_KEY")

	provider, err := openrouter.New(openrouter.WithAPIKey(apiKey))
	cli.ExitOnError(err, "Error creating provider")

	ctx := context.Background()

	model, err := provider.LanguageModel(ctx, "openai/gpt-4o")
	cli.ExitOnError(err, "Error getting model")

	analyzer := vision.NewScreenshotAnalyzer(model).
		WithSystemPrompt("You are a senior frontend engineer reviewing UI designs.")

	fmt.Println("Analyzing screenshot...")

	result, err := analyzer.AnalyzeScreenshot(
		ctx,
		"Review this UI for accessibility and design issues.",
		os.Args[1],
	)
	cli.ExitOnError(err, "")

	cli.PrintResult(result.Text, result.Usage.TotalTokens)
}
