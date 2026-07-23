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

	"github.com/larsartmann/vision-review-agent/internal/cli"
)

func main() {
	cli.RequireArgc(2)

	ctx := context.Background()
	model := cli.NewOpenAIModel(ctx, "gpt-4o")

	agent, err := cli.NewAgent(
		model,
		"You are a UI/UX expert. Analyze screenshots and provide actionable feedback.",
	)
	cli.ExitOnError(err, "Error creating agent")

	img := cli.LoadImageArg()

	fmt.Println("Analyzing screenshot...")

	result, err := agent.Analyze(ctx, "Describe the UI and identify any usability issues.", img)
	cli.ExitOnError(err, "")

	cli.PrintResult(result.Text, result.Usage.TotalTokens)
}
