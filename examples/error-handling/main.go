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
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

func main() {
	cli.RequireArgc(2)

	ctx := context.Background()
	model := cli.NewOpenAIModel(ctx, "gpt-4o")

	rc := vision.DefaultRetryConfig()
	rc.InitialBackoff = time.Second

	agent, err := vision.NewAgent(vision.Config{ //nolint:exhaustruct // optional fields use zero-value defaults
		Model:  model,
		Retry:  &rc,
		Temperature: 0.3,
	})
	cli.ExitOnError(err, "Error creating agent")

	img := cli.LoadImageArg()

	result, err := agent.Analyze(ctx, "Find the top UI issues in this screenshot.", img)
	if err != nil {
		handleError(err)
		return
	}

	cli.PrintResult(result.Text, result.Usage.TotalTokens)
}

// handleError demonstrates the switch-on-Kind pattern: each ErrorKind maps to
// a specific consumer action (retry, fix credentials, reduce input, etc.).
func handleError(err error) {
	modelErr, ok := errors.AsType[*vision.ModelError](err)
	if !ok {
		log.Fatalf("unexpected non-model error: %v", err)
	}

	switch modelErr.Kind {
	case vision.KindRateLimited:
		fmt.Println("Rate limited by the provider. Backing off and retrying later.")
	case vision.KindTimeout:
		fmt.Println("Request timed out. Try increasing the timeout or using smaller images.")
	case vision.KindServerError, vision.KindServiceUnavailable:
		fmt.Println("Provider is temporarily down. Retry with backoff.")
	case vision.KindNetwork:
		fmt.Println("Network error. Check your connection and retry.")
	case vision.KindAuthentication:
		fmt.Println("Authentication failed. Check your API key environment variable.")
	case vision.KindNotFound:
		fmt.Println("Model not found. Verify the model name in your config.")
	case vision.KindBadRequest:
		fmt.Println("Bad request. Check your prompt and image format.")
	case vision.KindContentFilter:
		fmt.Println("Content policy rejection. Modify your prompt or image.")
	case vision.KindContextTooLarge:
		fmt.Println("Input too large. Reduce the number or size of images.")
	case vision.KindNotImplemented:
		fmt.Println("Feature not implemented by this provider/model. Try a different model.")
	case vision.KindCancelled:
		fmt.Println("Request was cancelled.")
	case vision.KindStructuredParse:
		fmt.Println("Structured output parsing failed. Simplify your schema or prompt.")
	default:
		fmt.Printf("Unknown error kind %q: %v\n", modelErr.Kind, modelErr.Cause)
	}

	log.Fatalf("details: %v", err)
}
