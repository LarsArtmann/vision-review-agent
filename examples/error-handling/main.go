// Example: Error handling with classified model errors and enriched validation errors
//
// Demonstrates two error categories produced by the vision SDK:
//
//  1. Config validation errors — sentinel errors (ErrInvalidTemperature, etc.)
//     wrapped with the offending value for self-diagnosis. Use errors.Is to
//     match the sentinel; the message includes what value was wrong.
//
//  2. Model invocation errors — classified as *vision.ModelError with an
//     ErrorKind (rate-limited, timeout, bad-request, etc.). Use
//     errors.AsType to extract the kind and decide retry vs. fix-input.
//
// Usage:
//
//	export OPENAI_API_KEY=your-key
//	go run examples/error-handling/main.go screenshot.png
package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

func main() {
	ctx, model := cli.NewCLIContext(2)

	rc := vision.DefaultRetryConfig()
	rc.InitialBackoff = time.Second

	agent, err := vision.NewAgent(vision.Config{
		Model:       model,
		Retry:       &rc,
		Temperature: 0.3,
	})
	if err != nil {
		printConfigError(err)
		os.Exit(1)
	}

	img := cli.LoadImageArg()

	result, err := agent.Analyze(ctx, "Find the top UI issues in this screenshot.", img)
	if err != nil {
		printModelError(err)
		os.Exit(1)
	}

	cli.PrintResult(result.Text, result.Usage.TotalTokens)
}

// printConfigError handles config validation errors. These are sentinel errors
// wrapped with the offending value (e.g. "got 5.00, want [0.0, 2.0]").
// errors.Is still matches the underlying sentinel, and the message tells the
// user exactly what value to fix. This function does not call os.Exit.
func printConfigError(err error) {
	switch {
	case errors.Is(err, vision.ErrNoModel):
		fmt.Fprintln(os.Stderr, "No model configured. Set the Model field in Config.")
	case errors.Is(err, vision.ErrInvalidTemperature):
		fmt.Fprintf(os.Stderr, "Invalid temperature: %v\n", err)
	case errors.Is(err, vision.ErrInvalidMaxTokens):
		fmt.Fprintf(os.Stderr, "Invalid max output tokens: %v\n", err)
	case errors.Is(err, vision.ErrInvalidTopP):
		fmt.Fprintf(os.Stderr, "Invalid top-p: %v\n", err)
	case errors.Is(err, vision.ErrInvalidTopK):
		fmt.Fprintf(os.Stderr, "Invalid top-k: %v\n", err)
	case errors.Is(err, vision.ErrInvalidPresencePenalty):
		fmt.Fprintf(os.Stderr, "Invalid presence penalty: %v\n", err)
	case errors.Is(err, vision.ErrInvalidFrequencyPenalty):
		fmt.Fprintf(os.Stderr, "Invalid frequency penalty: %v\n", err)
	default:
		fmt.Fprintf(os.Stderr, "Agent creation failed: %v\n", err)
	}
}

// printModelError demonstrates the map-lookup-on-Kind pattern: it extracts the
// classified ModelError, prints actionable advice keyed by Kind, then prints the
// underlying cause for debugging. It does not call os.Exit — the caller decides
// the exit code, keeping this function reusable and testable.
func printModelError(err error) {
	modelErr, ok := errors.AsType[*vision.ModelError](err)
	if !ok {
		fmt.Fprintf(os.Stderr, "unexpected non-model error: %v\n", err)

		return
	}

	adviceByKind := map[vision.ErrorKind]string{
		vision.KindRateLimited:        "Rate limited by the provider. Backing off and retrying later.",
		vision.KindTimeout:            "Request timed out. Try increasing the timeout or using smaller images.",
		vision.KindServerError:        "Provider is temporarily down. Retry with backoff.",
		vision.KindServiceUnavailable: "Provider is temporarily unavailable. Retry with backoff.",
		vision.KindNetwork:            "Network error. Check your connection and retry.",
		vision.KindAuthentication:     "Authentication failed. Check your API key environment variable.",
		vision.KindNotFound:           "Model not found. Verify the model name in your config.",
		vision.KindBadRequest:         "Bad request. Check your prompt and image format.",
		vision.KindContentFilter:      "Content policy rejection. Modify your prompt or image.",
		vision.KindContextTooLarge:    "Input too large. Reduce the number or size of images.",
		vision.KindNotImplemented:     "Feature not implemented. Try a different model.",
		vision.KindCancelled:          "Request was cancelled.",
		vision.KindStructuredParse:    "Structured output parsing failed. Simplify your schema or prompt.",
		vision.KindUnknown:            "Unclassified error.",
	}

	advice := adviceByKind[modelErr.Kind]
	fmt.Fprintf(os.Stderr, "[%s] %s\n", modelErr.Kind, advice)
	fmt.Fprintf(os.Stderr, "  details: %v\n", modelErr.Cause)
}
