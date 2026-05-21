// Package cli provides shared helpers for CLI tools.
package cli

import (
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
