// Example: Error handling with classified model errors
//
// Demonstrates the errors.AsType[*vision.ModelError] pattern for inspecting
// model invocation failures by ErrorKind, enabling retry vs. fix-input
// decisions without reaching into the underlying provider SDK.
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
	cli.ExitOnError(err, "Error creating agent")

	img := cli.LoadImageArg()

	result, err := agent.Analyze(ctx, "Find the top UI issues in this screenshot.", img)
	if err != nil {
		printModelError(err)
		os.Exit(1)
	}

	cli.PrintResult(result.Text, result.Usage.TotalTokens)
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
