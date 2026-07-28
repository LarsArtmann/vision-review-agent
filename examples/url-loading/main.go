// Example: Loading images from a URL and from base64
//
// Demonstrates LoadImageFromURL (with an optional custom HTTP client) and
// LoadImageFromBase64.
//
// Usage:
//
//	export OPENAI_API_KEY=your-key
//	go run examples/url-loading/main.go https://example.com/screenshot.png
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

func main() {
	cli.RequireArgc(2)

	ctx := context.Background()
	model := cli.NewOpenAIModel(ctx, "gpt-4o")

	agent, err := cli.NewAgent(model, "You are a UI/UX expert. Describe screenshots.", 0.3)
	cli.ExitOnError(err, "Error creating agent")

	url := os.Args[1]

	// Custom client with a timeout; pass nil to use http.DefaultClient.
	client := &http.Client{Timeout: 30 * time.Second} //nolint:exhaustruct // only Timeout is relevant here

	img, err := vision.LoadImageFromURLWithClient(ctx, url, client)
	cli.ExitOnError(err, "Error loading image from URL")

	fmt.Printf("Loaded %q (%d bytes, %s)\n", img.Filename, len(img.Data), img.MediaType)

	result, err := agent.Analyze(ctx, "Describe this UI and list any issues.", img)
	cli.ExitOnError(err, "")

	cli.PrintResult(result.Text, result.Usage.TotalTokens)

	demonstrateBase64(img)
}

// demonstrateBase64 shows the round-trip: encode to base64, decode, and confirm.
func demonstrateBase64(img *vision.ImageSource) {
	encoded := base64.StdEncoding.EncodeToString(img.Data)

	decoded, err := vision.LoadImageFromBase64(encoded, img.MediaType, img.Filename)
	cli.ExitOnError(err, "Error decoding base64")

	fmt.Printf("\nBase64 round-trip OK: %d bytes -> %d base64 chars -> %d bytes\n",
		len(img.Data), len(encoded), len(decoded.Data))
}
