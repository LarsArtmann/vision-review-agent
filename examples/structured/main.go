// Example: Structured output from vision analysis
//
// This demonstrates how to get typed, structured results from image analysis
// instead of free-form text.
//
// Usage:
//
//	export OPENAI_API_KEY=your-key
//	go run examples/structured/main.go screenshot.png
package main

import (
	"context"
	"fmt"
	"os"

	"charm.land/fantasy/providers/openai"
	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

// UIReview is the structured output type.
type UIReview struct {
	Layout      string   `json:"layout"      description:"Brief description of the overall layout"`
	Components  []string `json:"components"  description:"List of UI components identified"`
	Issues      []Issue  `json:"issues"      description:"List of issues found"`
	Score       int      `json:"score"       description:"Overall UX score from 1-10"`
	Suggestions []string `json:"suggestions" description:"Actionable improvement suggestions"`
}

// Issue represents a single UI issue.
type Issue struct {
	Severity    string `json:"severity"    description:"Severity: critical, major, minor, or info"`
	Component   string `json:"component"   description:"Which component has the issue"`
	Description string `json:"description" description:"Detailed description of the issue"`
}

func main() {
	cli.RequireArgc(2)

	apiKey := cli.RequireEnvVar("OPENAI_API_KEY")

	provider, err := openai.New(openai.WithAPIKey(apiKey))
	cli.ExitOnError(err, "Error creating provider")

	ctx := context.Background()
	model, err := provider.LanguageModel(ctx, "gpt-4o")
	cli.ExitOnError(err, "Error getting model")

	agent, err := vision.NewAgent(vision.Config{
		SystemPrompt: "You are a meticulous UI/UX reviewer. Analyze designs thoroughly.",
		Model:        model,
		Temperature:  0.2,
	})
	cli.ExitOnError(err, "Error creating agent")

	img, err := vision.LoadImageFromFile(os.Args[1])
	cli.ExitOnError(err, "Error loading image")

	fmt.Println("Analyzing screenshot with structured output...")
	result, err := vision.AnalyzeStructured[UIReview](
		ctx,
		agent,
		"Review this UI design comprehensively.",
		img,
	)
	cli.ExitOnError(err, "Error during analysis")

	review := result.Object

	fmt.Printf("\n=== UX Score: %d/10 ===\n\n", review.Score)

	fmt.Println("Layout:", review.Layout)
	fmt.Println("\nComponents:")
	for _, c := range review.Components {
		fmt.Printf("  - %s\n", c)
	}

	fmt.Println("\nIssues:")
	for _, issue := range review.Issues {
		fmt.Printf("  [%s] %s: %s\n", issue.Severity, issue.Component, issue.Description)
	}

	fmt.Println("\nSuggestions:")
	for _, s := range review.Suggestions {
		fmt.Printf("  - %s\n", s)
	}

	fmt.Printf("\nTokens used: %d\n", result.Usage.TotalTokens)
}
