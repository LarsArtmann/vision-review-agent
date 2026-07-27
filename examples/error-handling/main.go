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
		Model:       model,
		Retry:       &rc,
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

// handleError demonstrates the map-lookup-on-Kind pattern: each ErrorKind maps
// to a specific consumer action (retry, fix credentials, reduce input, etc.).
func handleError(err error) {
	modelErr, ok := errors.AsType[*vision.ModelError](err)
	if !ok {
		log.Fatalf("unexpected non-model error: %v", err)
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

	advice, found := adviceByKind[modelErr.Kind]
	if !found {
		fmt.Printf("Unhandled error kind %q: %v\n", modelErr.Kind, modelErr.Cause)

		log.Fatalf("details: %v", err)
	}

	fmt.Println(advice)

	log.Fatalf("details: %v", err)
}
