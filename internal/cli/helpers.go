// Package cli provides shared helpers for CLI tools.
package cli

import (
	"context"
	"fmt"
	"os"

	"charm.land/fantasy"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

// ExitOnError prints the error message and exits with code 1 if err is not nil.
// If msg is non-empty, it is prefixed to the error.
func ExitOnError(err error, msg string) {
	if err == nil {
		return
	}

	if msg != "" {
		fmt.Fprintln(os.Stderr, msg+":", err)
	} else {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}

	os.Exit(1)
}

// RequireArgc checks if the given argc is at least minArgs.
// If not, it prints the usage message and exits with code 1.
func RequireArgc(minArgs int) {
	if len(os.Args) < minArgs {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0])
		os.Exit(1)
	}
}

// RequireEnvVar checks if the environment variable is set.
// If not, it prints a message and exits with code 1.
func RequireEnvVar(name string) string {
	value := os.Getenv(name)
	if value == "" {
		fmt.Fprintf(os.Stderr, "Please set %s\n", name)
		os.Exit(1)
	}

	return value
}

// PrintResult prints the analysis result in a consistent format.
func PrintResult(text string, totalTokens int64) {
	fmt.Println("\n--- Analysis ---")
	fmt.Println(text)
	fmt.Printf("\nTokens used: %d\n", totalTokens)
}

// NewAgent creates a new vision agent with the given model and system prompt.
// Temperature defaults to 0.3 if not specified.
func NewAgent(
	model fantasy.LanguageModel,
	systemPrompt string,
	temperature ...float64,
) (*vision.Agent, error) {
	temp := 0.3
	if len(temperature) > 0 {
		temp = temperature[0]
	}

	//nolint:exhaustruct // optional fields intentionally use zero-value defaults
	agent, err := vision.NewAgent(vision.Config{
		SystemPrompt: systemPrompt,
		Model:        model,
		Temperature:  temp,
	})
	if err != nil {
		return nil, fmt.Errorf("create vision agent (systemPrompt=%q): %w", systemPrompt, err)
	}

	return agent, nil
}

// NewCLIContext validates the minimum argument count, creates a background
// context, and constructs the default OpenAI vision model (gpt-4o). It is the
// shared bootstrap prologue for CLI examples and tools that need a custom
// Config; prefer NewAgentFromArgs for the common case.
//
// RequireArgc panics (exits) if fewer than minArgs are present, so the returned
// context and model are always usable.
func NewCLIContext(minArgs int) (context.Context, fantasy.LanguageModel) {
	RequireArgc(minArgs)

	ctx := context.Background()

	return ctx, NewOpenAIModel(ctx, "gpt-4o")
}

// NewAgentFromArgs is the one-line bootstrap for examples and CLI tools that
// use the default gpt-4o model with a system prompt. It validates the argument
// count, builds the context and model, constructs the agent, and exits on
// construction error — returning a ready-to-use (ctx, agent) pair.
//
// Examples that need a custom Config (Hooks, Retry, etc.) should call
// NewCLIContext and vision.NewAgent directly.
func NewAgentFromArgs(
	minArgs int,
	systemPrompt string,
	temperature ...float64,
) (context.Context, *vision.Agent) {
	ctx, model := NewCLIContext(minArgs)
	agent, err := NewAgent(model, systemPrompt, temperature...)
	ExitOnError(err, "Error creating agent")

	return ctx, agent
}

// LoadImageArg loads an image from the first positional CLI argument
// (os.Args[1]). It exits the process on any error. RequireArgc(2) must be
// called first so os.Args[1] exists.
func LoadImageArg() *vision.ImageSource {
	img, err := vision.LoadImageFromFile(os.Args[1])
	ExitOnError(err, "Error loading image")

	return img
}

// AnalyzeAndPrint is the one-call workflow for examples that load an image from
// the first CLI argument, run a single analysis, and print the result. It
// collapses LoadImageArg → Analyze → ExitOnError → PrintResult so the simplest
// examples read as a single expressive line. Examples that need custom error
// handling, hooks, or non-standard image sources should call the individual
// helpers instead.
func AnalyzeAndPrint(ctx context.Context, agent *vision.Agent, prompt string) {
	img := LoadImageArg()
	result, err := agent.Analyze(ctx, prompt, img)
	ExitOnError(err, "")
	PrintResult(result.Text, result.Usage.TotalTokens)
}
