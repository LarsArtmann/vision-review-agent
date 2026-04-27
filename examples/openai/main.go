// Example: OpenAI vision analysis
//
// Usage:
//
//	export OPENAI_API_KEY=your-key
//	go run examples/openai/main.go screenshot.png
package main

import (
	"context"
	"fmt"
	"os"

	"charm.land/fantasy/providers/openai"
	"github.com/larsartmann/vision-review-agent/vision"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: go run examples/openai/main.go <screenshot.png>")
		os.Exit(1)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Please set OPENAI_API_KEY")
		os.Exit(1)
	}

	provider, err := openai.New(openai.WithAPIKey(apiKey))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating provider:", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Use gpt-4o for vision support
	model, err := provider.LanguageModel(ctx, "gpt-4o")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting model:", err)
		os.Exit(1)
	}

	// Create agent with custom system prompt
	agent, err := vision.NewAgent(vision.Config{
		SystemPrompt: "You are a UI/UX expert. Analyze screenshots and provide actionable feedback.",
		Model:        model,
		Temperature:  0.3,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating agent:", err)
		os.Exit(1)
	}

	// Load image
	img, err := vision.LoadImageFromFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error loading image:", err)
		os.Exit(1)
	}

	// Analyze
	fmt.Println("Analyzing screenshot...")
	result, err := agent.Analyze(ctx, "Describe the UI and identify any usability issues.", img)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	fmt.Println("\n--- Analysis ---")
	fmt.Println(result.Text)
	fmt.Printf("\nTokens used: %d\n", result.Usage.TotalTokens)
}
