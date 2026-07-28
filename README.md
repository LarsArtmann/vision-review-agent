# Vision Review Agent

A simple, production-ready Go SDK for building AI agents with vision capabilities. Built on top of [charm.land/fantasy](https://github.com/charmbracelet/fantasy).

## Features

- **Simple API** — Analyze images/screenshots with a single function call
- **Multi-provider** — Works with OpenAI, OpenRouter, Anthropic, Google, Azure, Bedrock, and any fantasy-compatible provider
- **Streaming** — Stream responses in real-time (text and structured)
- **Structured Output** — Get typed, structured results instead of free-form text
- **Structured Streaming** — Stream partial structured objects as they arrive
- **Multi-turn Conversations** — Maintain conversation history for follow-up questions
- **Batch Analysis** — Analyze multiple images concurrently with bounded parallelism
- **Lifecycle Hooks** — Observe analysis lifecycle via OnStart, OnFinish, OnError callbacks
- **Image Preprocessing** — Auto-resize and compress before analysis to cut token cost
- **Automatic Retry** — Optional vision-layer retry with backoff + jitter on transient errors
- **Cost Tracking** — Thread-safe token accumulator wired into hooks
- **Flexible Image Loading** — Load from files, URLs, base64 strings, or any `io.Reader`
- **Classified Errors** — Every model error is classified into 14 kinds with `IsRetryable()` for smart retry logic
- **Built-in CLI** — Analyze images from the command line
- **Full Model Parameters** — Temperature, TopP, TopK, PresencePenalty, FrequencyPenalty
- **Validation** — Strong input validation with clear error types
- **Configurable** — Token limits, retries, timeouts, sampling parameters

## Installation

```bash
go get github.com/larsartmann/vision-review-agent
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"os"

	"charm.land/fantasy/providers/openai"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

func main() {
	provider, _ := openai.New(openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")))
	model, _ := provider.LanguageModel(context.Background(), "gpt-4o")

	agent, _ := vision.NewAgent(vision.Config{
		SystemPrompt: "You are a UI/UX expert.",
		Model:        model,
	})

	img, _ := vision.LoadImageFromFile("screenshot.png")
	result, _ := agent.Analyze(context.Background(), "Find UI bugs", img)

	fmt.Println(result.Text)
}
```

## Development

This repo uses [Nix flakes](flake.nix) for reproducible builds. There is no
`justfile`/`Makefile` — use `go` directly or the flake apps.

```bash
# Go toolchain (fast iteration)
go build ./...            # Build everything
go test -race ./...       # Run all tests with race detector
go vet ./...              # Run go vet
golangci-lint run ./...   # Lint

# Nix flake (reproducible)
nix develop               # Enter dev shell
nix run .#test            # go test -race -v -coverprofile=coverage.out ./...
nix run .#lint            # golangci-lint run ./...
nix build .               # Build the package
nix flake check           # Canonical quality gate
```

## CLI Usage

```bash
# Build the CLI
go build -o vision ./cmd/vision

# Analyze a screenshot
export OPENAI_API_KEY=your-key
./vision -prompt "Find UI bugs" screenshot.png

# Stream the response
./vision -stream -prompt "Describe this UI" screenshot.png

# Use OpenRouter
export OPENROUTER_API_KEY=your-key
./vision -provider openrouter -model anthropic/claude-3.5-sonnet screenshot.png

# Structured JSON output (built-in UI review schema)
./vision -structured -prompt "Review this UI" screenshot.png

# Custom system prompt + timeout
./vision -system "You are an accessibility expert" -prompt "Check WCAG compliance" -timeout 30 screenshot.png
```

CLI flags: `-provider`, `-model`, `-prompt`, `-system`, `-stream`,
`-temperature`, `-max-tokens`, `-json`, `-structured`, `-timeout`, `-version`.

## SDK Usage

### Basic Analysis

```go
agent, err := vision.NewAgent(vision.Config{
	SystemPrompt: "Describe what you see.",
	Model:        model,
	Temperature:  0.3,
})
if err != nil {
	log.Fatal(err)
}

img, _ := vision.LoadImageFromFile("screenshot.png")
result, _ := agent.Analyze(ctx, "What UI issues do you see?", img)
fmt.Println(result.Text)
```

### Streaming Analysis

```go
result, _ := agent.AnalyzeStream(ctx, "Describe this UI", func(text string) error {
	fmt.Print(text) // Print each chunk as it arrives
	return nil
}, img)
```

### Structured Output

```go
type UIReview struct {
	Layout      string   `json:"layout"`
	Issues      []string `json:"issues"`
	Score       int      `json:"score"`
	Suggestions []string `json:"suggestions"`
}

result, _ := vision.AnalyzeStructured[UIReview](ctx, agent, "Review this UI", img)
fmt.Printf("Score: %d/10\n", result.Object.Score)
```

### Image Preprocessing

Auto-resize and compress images before every analysis to cut token cost and
stay within provider dimension limits:

```go
agent, _ := vision.NewAgent(vision.Config{
	Model: model,
	Preprocess: &vision.PreprocessConfig{
		MaxDimension: 1568, // OpenAI's recommended max
		JPEGQuality:  75,   // 1-100; 0 defaults to 85
	},
})

// Images are auto-resized on every Analyze* call. Resize/compress manually too:
small, _ := vision.ResizeImage(img, 1568)
smaller, _ := vision.CompressImage(img, 60) // re-encode JPEG at lower quality
```

### Automatic Retry

`Config.Retry` enables vision-layer retry of transient (`IsRetryable`) failures
across all non-streaming analysis methods, with exponential backoff + jitter:

```go
rc := vision.DefaultRetryConfig() // 3 attempts, backoff, jitter
rc.InitialBackoff = time.Second

agent, _ := vision.NewAgent(vision.Config{
	Model: model,
	Retry: &rc,
})

// Analyze now retries automatically on rate limits, server errors, etc.
// Streaming methods do NOT auto-retry (ambiguous delta semantics); wrap them
// in vision.WithRetry manually if needed.
```

This composes with `Config.MaxRetries` (fantasy's HTTP-layer retry) for layered
retry if both are set.

### Cost Tracking

`NewAgentWithCostTracker` auto-wires a thread-safe `CostTracker` into
`Hooks.OnFinish`, composing with any user-supplied hooks:

```go
agent, tracker, err := vision.NewAgentWithCostTracker(vision.Config{Model: model})
if err != nil {
	log.Fatal(err)
}

result, _ := agent.Analyze(ctx, "describe", img)

total := tracker.Total() // fantasy.Usage{InputTokens, OutputTokens, TotalTokens}
fmt.Printf("%d calls, %d total tokens\n", tracker.Calls(), total.TotalTokens)
```

### ScreenshotAnalyzer Builder

```go
analyzer := vision.NewScreenshotAnalyzer(model).
	WithSystemPrompt("Find accessibility issues").
	WithTemperature(0.2).
	WithTopP(0.9).
	WithMaxOutputTokens(1000).
	WithMaxDimension(1568) // sets Config.Preprocess

result, _ := analyzer.AnalyzeScreenshot(ctx, "Review this page", "screenshot.png")
```

### Multi-turn Conversations

```go
conv := vision.NewConversation()
conv.AddUserMessage("Describe this UI", img)

result, _ := agent.AnalyzeConversation(ctx, conv, "What UI issues do you see?", img)
conv.AddAssistantMessage(result.Text)

// Follow-up with full context
followUp, _ := agent.AnalyzeConversation(ctx, conv, "What about the color contrast?", img)
```

### Batch Analysis

```go
images := []*vision.ImageSource{img1, img2, img3}
results := agent.AnalyzeBatch(ctx, "Describe", 3, images...)

for _, r := range results {
	if r.Err != nil {
		log.Printf("image %d failed: %v", r.Index, r.Err)
		continue
	}
	fmt.Println(r.Index, r.Result.Text)
}
```

### Lifecycle Hooks

```go
agent, err := vision.NewAgent(vision.Config{
	Model: model,
	Hooks: vision.Hooks{
		OnStart: func(_ context.Context, _ string, n int) {
			log.Printf("analyzing %d images", n)
		},
		OnFinish: func(_ context.Context, result *vision.AnalyzeResult) {
			log.Printf("used %d tokens", result.Usage.TotalTokens)
		},
		OnError: func(_ context.Context, err error) {
			log.Printf("failed: %v", err)
		},
	},
})
```

### Classified Errors with Retry Logic

Every model error is wrapped in a `*vision.ModelError` with an `ErrorKind` and
`IsRetryable()`. Extract it with Go 1.26+'s generic `errors.AsType`:

```go
result, err := agent.Analyze(ctx, "describe", img)
if err != nil {
	modelErr, ok := errors.AsType[*vision.ModelError](err)
	if ok && modelErr.IsRetryable() {
		// back off and retry (or rely on Config.Retry to do it for you)
	}
	switch modelErr.Kind {
	case vision.KindRateLimited:    // ...
	case vision.KindAuthentication: // ...
	}
}
```

See [`examples/error-handling/`](examples/error-handling/) for a complete
kind-to-action lookup example.

### Flexible Image Loading

```go
// From file
img, _ := vision.LoadImageFromFile("screenshot.png")

// From URL (validates magic bytes; rejects non-image bodies)
img, _ := vision.LoadImageFromURL(ctx, "https://example.com/screenshot.png")

// With a custom HTTP client (proxies, timeouts, TLS)
img, _ := vision.LoadImageFromURLWithClient(ctx, url, httpClient)

// From base64 (supports standard, URL-safe, and raw encodings)
img, _ := vision.LoadImageFromBase64(b64String, vision.MediaTypePNG, "capture.png")

// From any io.Reader
img, _ := vision.LoadImageFromReader(file, vision.MediaTypeJPEG, "photo.jpg")
```

## Configuration

| Option             | Description                                                              |
| ------------------ | ------------------------------------------------------------------------ |
| `Model`            | The LLM to use (must support vision) — **required**                      |
| `SystemPrompt`     | Defines agent behavior                                                   |
| `Temperature`      | Randomness (0.0-2.0)                                                     |
| `TopP`             | Nucleus sampling (0.0-1.0)                                               |
| `TopK`             | Top-k sampling limit                                                     |
| `PresencePenalty`  | Penalize tokens already present (-2 to 2)                                |
| `FrequencyPenalty` | Penalize tokens by frequency (-2 to 2)                                   |
| `MaxOutputTokens`  | Response length limit                                                    |
| `MaxRetries`       | Fantasy HTTP-layer retry count for transient errors (0 disables)         |
| `Retry`            | Vision-layer `*RetryConfig` (backoff + jitter) for non-streaming methods |
| `RequestTimeout`   | Per-request timeout                                                      |
| `Preprocess`       | `*PreprocessConfig` (auto-resize + JPEG quality) before every Analyze*   |
| `Hooks`            | Lifecycle callbacks (OnStart/OnFinish/OnError)                           |

## Error Types

**Validation errors** (returned before any model call):

- `ErrNoModel` — No model configured
- `ErrEmptyPrompt` — Empty prompt provided
- `ErrNoImages` — No images provided
- `ErrInvalidTemperature` — Temperature out of range
- `ErrInvalidMaxTokens` — Negative max tokens
- `ErrInvalidTopP` — Top-p out of 0.0-1.0 range
- `ErrInvalidTopK` — Negative top-k
- `ErrInvalidPresencePenalty` — Presence penalty out of -2.0 to 2.0
- `ErrInvalidFrequencyPenalty` — Frequency penalty out of -2.0 to 2.0
- `ErrInvalidImage` — Image data does not match known format
- `ErrEmptyImageData` — Image data is empty
- `ErrImageTooLarge` — Image exceeds 50 MB size limit

**Classified model errors** (`*vision.ModelError`, extract via `errors.AsType`):

| Kind                     | Retryable | Description                                  |
| ------------------------ | --------- | -------------------------------------------- |
| `KindRateLimited`        | Yes       | Provider returned 429                        |
| `KindTimeout`            | Yes       | Request exceeded deadline                    |
| `KindServerError`        | Yes       | Provider returned 5xx (not 501/503)          |
| `KindServiceUnavailable` | Yes       | Provider returned 503                        |
| `KindNetwork`            | Yes       | Transport-level failure                      |
| `KindAuthentication`     | No        | Invalid credentials (401/403)                |
| `KindNotFound`           | No        | Model or resource not found (404)            |
| `KindBadRequest`         | No        | Provider rejected request (400)              |
| `KindContentFilter`      | No        | Content policy rejection (detected from 400) |
| `KindNotImplemented`     | No        | Provider returned 501                        |
| `KindContextTooLarge`    | No        | Input exceeded context window                |
| `KindCancelled`          | No        | Context was cancelled                        |
| `KindStructuredParse`    | No        | JSON parse failure in structured mode        |
| `KindUnknown`            | No        | Unclassified                                 |

## Examples

See the [`examples/`](examples/) directory for complete working examples:

- [`examples/openai/`](examples/openai/) — OpenAI provider
- [`examples/openrouter/`](examples/openrouter/) — OpenRouter provider
- [`examples/structured/`](examples/structured/) — Structured output
- [`examples/structured-stream/`](examples/structured-stream/) — Structured streaming
- [`examples/conversation/`](examples/conversation/) — Multi-turn conversations
- [`examples/batch/`](examples/batch/) — Batch analysis
- [`examples/hooks/`](examples/hooks/) — Lifecycle hooks
- [`examples/url-loading/`](examples/url-loading/) — Loading from URLs
- [`examples/error-handling/`](examples/error-handling/) — Classified error handling

## Project Structure

```
pkg/
  vision/         Public SDK (Agent, Config, image loading, analysis, preprocessing)
  errors/         Centralized domain errors (re-exported from pkg/vision)
internal/
  visionutil/     Internal helpers (prompt building, unmarshaling)
cmd/vision/       CLI tool
examples/         Working examples for each provider/feature
```

## Domain Language

For the ubiquitous language, bounded contexts, and entity/value-object
glossary used throughout this project, see
[docs/DOMAIN_LANGUAGE.md](docs/DOMAIN_LANGUAGE.md).

## License

Proprietary — see [LICENSE](LICENSE) for details.
