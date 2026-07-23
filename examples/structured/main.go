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

	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

// UIReview is the structured output type.
type UIReview struct {
	Layout      string   `description:"Brief description of the overall layout" json:"layout"`
	Components  []string `description:"List of UI components identified"        json:"components"`
	Issues      []Issue  `description:"List of issues found"                    json:"issues"`
	Score       int      `description:"Overall UX score from 1-10"              json:"score"`
	Suggestions []string `description:"Actionable improvement suggestions"      json:"suggestions"`
}

// Issue represents a single UI issue.
type Issue struct {
	Severity    string `description:"Severity: critical, major, minor, or info" json:"severity"`
	Component   string `description:"Which component has the issue"             json:"component"`
	Description string `description:"Detailed description of the issue"         json:"description"`
}

func main() {
	cli.RequireArgc(2)

	ctx := context.Background()
	model := cli.NewOpenAIModel(ctx, "gpt-4o")

	agent, err := cli.NewAgent(
		model,
		"You are a meticulous UI/UX reviewer. Analyze designs thoroughly.",
		0.2,
	)
	cli.ExitOnError(err, "Error creating agent")

	img := cli.LoadImageArg()

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
