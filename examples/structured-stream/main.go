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
	"fmt"

	"github.com/larsartmann/vision-review-agent/examples/internal/uireview"
	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

func main() {
	ctx, agent := cli.NewAgentFromArgs(2, "You are a meticulous UI/UX reviewer.", 0.2)

	onPartial := func(partial uireview.UIReview) {
		fmt.Printf("  [partial] score=%d, components=%d, issues=%d\n",
			partial.Score, len(partial.Components), len(partial.Issues))
	}

	fmt.Println("Streaming structured review (partials printed as they arrive)...")

	img := cli.LoadImageArg()

	result, err := vision.AnalyzeStructuredStream[uireview.UIReview](
		ctx,
		agent,
		"Review this UI design comprehensively.",
		onPartial,
		img,
	)
	cli.ExitOnError(err, "Error during streaming analysis")

	fmt.Printf("\n=== Final UX Score: %d/10 ===\n", result.Object.Score)
	fmt.Println("Layout:", result.Object.Layout)
	fmt.Printf("Tokens used: %d\n", result.Usage.TotalTokens)
}
