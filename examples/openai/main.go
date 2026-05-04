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
	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

func main() {
	cli.RequireArgc(2)

	apiKey := cli.RequireEnvVar("OPENAI_API_KEY")

	provider, err := openai.New(openai.WithAPIKey(apiKey))
	cli.ExitOnError(err, "Error creating provider")

	ctx := context.Background()

	model, err := provider.LanguageModel(ctx, "gpt-4o")
	cli.ExitOnError(err, "Error getting model")

	agent, err := cli.NewAgent(
		model,
		"You are a UI/UX expert. Analyze screenshots and provide actionable feedback.",
	)
	cli.ExitOnError(err, "Error creating agent")

	img, err := vision.LoadImageFromFile(os.Args[1])
	cli.ExitOnError(err, "Error loading image")

	fmt.Println("Analyzing screenshot...")
	result, err := agent.Analyze(ctx, "Describe the UI and identify any usability issues.", img)
	cli.ExitOnError(err, "")

	cli.PrintResult(result.Text, result.Usage.TotalTokens)
}
