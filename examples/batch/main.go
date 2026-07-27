// Example: Concurrent batch analysis
//
// Demonstrates analyzing many images concurrently with bounded parallelism.
// Results arrive in input order with per-image error handling.
//
// Usage:
//
//	export OPENAI_API_KEY=your-key
//	go run examples/batch/main.go screenshot1.png screenshot2.png screenshot3.png
package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

func main() {
	cli.RequireArgc(2)

	ctx := context.Background()
	model := cli.NewOpenAIModel(ctx, "gpt-4o")

	agent, err := cli.NewAgent(model, "You are a UI/UX expert. Analyze each screen briefly.", 0.3)
	cli.ExitOnError(err, "Error creating agent")

	images := loadAllImages()

	results := agent.AnalyzeBatch(ctx, "Summarize the key UI issues in this screen.", 2, images...)
	for _, result := range results {
		printBatchResult(result)
	}
}

func loadAllImages() []*vision.ImageSource {
	images := make([]*vision.ImageSource, 0, flag.NArg())

	for i := range flag.NArg() {
		path := flag.Arg(i)
		img, err := vision.LoadImageFromFile(path)
		cli.ExitOnError(err, "loading "+path)

		images = append(images, img)
	}

	return images
}

func printBatchResult(result vision.BatchResult) {
	fmt.Printf("\n=== Image %d ===\n", result.Index)

	if result.Err != nil {
		fmt.Printf("  failed: %v\n", result.Err)

		return
	}

	fmt.Printf("  %s\n", result.Result.Text)
	fmt.Printf("  tokens: %d\n", result.Result.Usage.TotalTokens)
}
