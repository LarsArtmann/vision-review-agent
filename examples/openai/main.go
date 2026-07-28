// Example: OpenAI vision analysis
//
// The simplest end-to-end example: bootstrap an agent, analyze one screenshot,
// and print the result.
//
// Usage:
//
//	export OPENAI_API_KEY=your-key
//	go run examples/openai/main.go screenshot.png
package main

import (
	"github.com/larsartmann/vision-review-agent/internal/cli"
)

func main() {
	ctx, agent := cli.NewAgentFromArgs(
		2,
		"You are a UI/UX expert. Analyze screenshots and provide actionable feedback.",
	)

	cli.AnalyzeAndPrint(ctx, agent, "Describe the UI and identify any usability issues.")
}
