// Example: Multi-turn conversation analysis
//
// Demonstrates follow-up questions that reference previous context using a
// vision.Conversation. Each turn persists the user message + assistant reply.
//
// Usage:
//
//	export OPENAI_API_KEY=your-key
//	go run examples/conversation/main.go screenshot.png
package main

import (
	"context"
	"fmt"

	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

func main() {
	cli.RequireArgc(2)

	ctx := context.Background()
	model := cli.NewOpenAIModel(ctx, "gpt-4o")

	agent, err := cli.NewAgent(
		model,
		"You are a UI/UX expert. Answer concisely and reference earlier context.",
		0.3,
	)
	cli.ExitOnError(err, "Error creating agent")

	img := cli.LoadImageArg()
	conv := vision.NewConversation()

	turn(ctx, agent, conv, img, "Describe the overall layout of this UI.")
	turn(ctx, agent, conv, img, "What about the color contrast of the buttons?")
	turn(ctx, agent, conv, img, "Suggest one high-impact improvement based on the above.")
}

// turn runs one conversational analysis and persists it into the history.
func turn(
	ctx context.Context,
	agent *vision.Agent,
	conv *vision.Conversation,
	img *vision.ImageSource,
	prompt string,
) {
	result, err := agent.AnalyzeConversation(ctx, conv, prompt, img)
	cli.ExitOnError(err, "")

	conv.AddUserMessage(prompt, img)
	conv.AddAssistantMessage(result.Text)

	fmt.Printf("\n--- %s ---\n", prompt)
	fmt.Println(result.Text)
	fmt.Printf("Tokens used: %d\n", result.Usage.TotalTokens)
}
