// Example: Structured streaming with partial objects
//
// Demonstrates AnalyzeStructuredStream, which emits typed partial objects as
// the model produces them — ideal for live UI updates.
//
// Usage:
//
//	export OPENAI_API_KEY=your-key
//	go run examples/structured-stream/main.go screenshot.png
package main

import (
	"context"
	"fmt"

	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

// UIReview is the structured output type produced incrementally.
type UIReview struct {
	Layout      string   `description:"Brief layout description"      json:"layout"`
	Components  []string `description:"UI components identified"       json:"components"`
	Issues      []Issue  `description:"Issues found"                   json:"issues"`
	Score       int      `description:"Overall UX score from 1-10"     json:"score"`
	Suggestions []string `description:"Improvement suggestions"        json:"suggestions"`
}

// Issue represents a single UI issue.
type Issue struct {
	Severity    string `description:"critical, major, minor, or info" json:"severity"`
	Component   string `description:"Affected component"              json:"component"`
	Description string `description:"Issue detail"                    json:"description"`
}

func main() {
	cli.RequireArgc(2)

	ctx := context.Background()
	model := cli.NewOpenAIModel(ctx, "gpt-4o")

	agent, err := cli.NewAgent(model, "You are a meticulous UI/UX reviewer.", 0.2)
	cli.ExitOnError(err, "Error creating agent")

	img := cli.LoadImageArg()

	fmt.Println("Streaming structured review (partials printed as they arrive)...")

	result, err := vision.AnalyzeStructuredStream[UIReview](
		ctx,
		agent,
		"Review this UI design comprehensively.",
		func(partial UIReview) {
			fmt.Printf("  [partial] score=%d, components=%d, issues=%d\n",
				partial.Score, len(partial.Components), len(partial.Issues))
		},
		img,
	)
	cli.ExitOnError(err, "Error during streaming analysis")

	fmt.Printf("\n=== Final UX Score: %d/10 ===\n", result.Object.Score)
	fmt.Println("Layout:", result.Object.Layout)
	fmt.Printf("Tokens used: %d\n", result.Usage.TotalTokens)
}
