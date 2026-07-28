// Example: Lifecycle hooks for logging and metrics
//
// Demonstrates OnStart / OnFinish / OnError callbacks fired synchronously by
// every analysis method. This example configures the agent directly (rather
// than via cli.NewAgent) to show the Config.Hooks field.
//
// Usage:
//
//	export OPENAI_API_KEY=your-key
//	go run examples/hooks/main.go screenshot.png
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

func main() {
	ctx, model := cli.NewCLIContext(2)

	agent, err := vision.NewAgent(vision.Config{
		Model:        model,
		SystemPrompt: "You are a UI/UX expert. Provide actionable feedback.",
		Temperature:  0.3,
		Hooks: vision.Hooks{
			OnStart: func(_ context.Context, prompt string, imageCount int) {
				log.Printf("start: %d image(s), prompt=%q", imageCount, prompt)
			},
			OnFinish: func(_ context.Context, result *vision.AnalyzeResult) {
				log.Printf("finish: %d tokens used", result.Usage.TotalTokens)
			},
			OnError: func(_ context.Context, failed error) {
				log.Printf("error: %v", failed)
			},
		},
	})
	cli.ExitOnError(err, "Error creating agent")

	fmt.Println("Analyzing (hooks log to stderr)...")

	cli.AnalyzeAndPrint(ctx, agent, "Find the top UI issues in this screenshot.")
}
