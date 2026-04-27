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
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: go run examples/openrouter/main.go <screenshot.png>")
		os.Exit(1)
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Please set OPENROUTER_API_KEY")
		os.Exit(1)
	}

	provider, err := openrouter.New(openrouter.WithAPIKey(apiKey))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating provider:", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Use a vision-capable model via OpenRouter
	model, err := provider.LanguageModel(ctx, "openai/gpt-4o")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting model:", err)
		os.Exit(1)
	}

	// Use the ScreenshotAnalyzer for convenience
	analyzer := vision.NewScreenshotAnalyzer(model).
		WithSystemPrompt("You are a senior frontend engineer reviewing UI designs.")

	// Analyze
	fmt.Println("Analyzing screenshot...")
	result, err := analyzer.AnalyzeScreenshot(
		ctx,
		"Review this UI for accessibility and design issues.",
		os.Args[1],
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	fmt.Println("\n--- Analysis ---")
	fmt.Println(result.Text)
	fmt.Printf("\nTokens used: %d\n", result.Usage.TotalTokens)
}
