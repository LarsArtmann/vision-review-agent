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
	"fmt"

	"github.com/larsartmann/vision-review-agent/examples/internal/uireview"
	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

func main() {
	ctx, agent := cli.NewAgentFromArgs(2, "You are a meticulous UI/UX reviewer. Analyze designs thoroughly.", 0.2)

	img := cli.LoadImageArg()

	fmt.Println("Analyzing screenshot with structured output...")

	result, err := vision.AnalyzeStructured[uireview.UIReview](
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
