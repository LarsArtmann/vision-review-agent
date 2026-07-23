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
- **Flexible Image Loading** — Load from files, URLs, base64 strings, or any `io.Reader`
- **Classified Errors** — Every model error is classified (rate-limited, auth, timeout, etc.) with `IsRetryable()` for smart retry logic
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

```bash
# Enter dev shell (requires Nix + direnv, or use `nix develop`)
direnv allow

# Available commands
just test        # Run tests
just coverage    # Run tests with 70% threshold
just test-race   # Run with race detector
just lint        # Run golangci-lint
just structure-lint  # Run go-structure-linter
just build       # Build all packages
just cli         # Build CLI binary
```

## CLI Usage

```bash
# Build the CLI
just cli

# Analyze a screenshot
export OPENAI_API_KEY=your-key
./vision-cli -prompt "Find UI bugs" screenshot.png

# Stream the response
./vision-cli -stream -prompt "Describe this UI" screenshot.png

# Use OpenRouter
export OPENROUTER_API_KEY=your-key
./vision-cli -provider openrouter -model anthropic/claude-3.5-sonnet screenshot.png

# Custom system prompt
./vision-cli -system "You are an accessibility expert" -prompt "Check WCAG compliance" screenshot.png
```

## SDK Usage

### Basic Analysis

```go
agent, err := vision.NewAgent(vision.Config{
    SystemPrompt: "Describe what you see.",
    Model:        model,
    Temperature:  0.3,
    MaxRetries:   3,
})

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

### ScreenshotAnalyzer Builder

```go
analyzer := vision.NewScreenshotAnalyzer(model).
    WithSystemPrompt("Find accessibility issues").
    WithTemperature(0.2).
    WithTopP(0.9).
    WithMaxOutputTokens(1000)

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

### Structured Streaming

```go
result, _ := vision.AnalyzeStructuredStream[UIReview](
    ctx, agent, "Review this UI",
    func(partial UIReview) {
        fmt.Printf("Score so far: %d\n", partial.Score) // real-time updates
    },
    img,
)
```

### Flexible Image Loading

```go
// From URL
img, _ := vision.LoadImageFromURL(ctx, "https://example.com/screenshot.png")

// From base64 (supports standard, URL-safe, and raw encodings)
img, _ := vision.LoadImageFromBase64(b64String, vision.MediaTypePNG, "capture.png")

// From any io.Reader
img, _ := vision.LoadImageFromReader(file, vision.MediaTypeJPEG, "photo.jpg")
```

### Lifecycle Hooks

```go
agent, _ := vision.NewAgent(vision.Config{
    Model: model,
    Hooks: vision.Hooks{
        OnStart:  func(ctx, prompt, n) { log.Printf("analyzing %d images", n) },
        OnFinish: func(ctx, result) { log.Printf("used %d tokens", result.Usage.TotalTokens) },
        OnError:  func(ctx, err) { log.Printf("failed: %v", err) },
    },
})
```

### Classified Errors with Retry Logic

```go
result, err := agent.Analyze(ctx, "describe", img)
if err != nil {
    var me *vision.ModelError
    if errors.AsType(err, &me) {
        if me.IsRetryable() {
            // back off and retry
        }
        switch me.Kind {
        case vision.KindRateLimited: // ...
        case vision.KindAuthentication: // ...
        }
    }
}
```

## Configuration

| Option            | Description                          |
| ----------------- | ------------------------------------ |
| `SystemPrompt`    | Defines agent behavior               |
| `Model`           | The LLM to use (must support vision) |
| `Temperature`     | Randomness (0.0-2.0)                 |
| `MaxOutputTokens` | Response length limit                |
| `MaxRetries`      | Retry count for transient errors     |
| `RequestTimeout`  | Per-request timeout                  |

## Error Types

- `ErrNoModel` — No model configured
- `ErrEmptyPrompt` — Empty prompt provided
- `ErrNoImages` — No images provided
- `ErrInvalidTemperature` — Temperature out of range
- `ErrInvalidMaxTokens` — Negative max tokens
- `ErrInvalidImage` — Image data does not match known format
- `ErrEmptyImageData` — Image data is empty
- `ErrImageTooLarge` — Image exceeds 50 MB size limit

## Examples

See the [`examples/`](examples/) directory for complete working examples:

- [`examples/openai/`](examples/openai/) — OpenAI provider
- [`examples/openrouter/`](examples/openrouter/) — OpenRouter provider
- [`examples/structured/`](examples/structured/) — Structured output

## Project Structure

```
pkg/
  vision/         Public SDK (Agent, Config, image loading, analysis)
  errors/         Centralized domain errors (re-exported from pkg/vision)
internal/
  visionutil/     Internal helpers (prompt building, unmarshaling)
cmd/vision/       CLI tool
examples/         Working examples for each provider
```

## License

Proprietary — see [LICENSE](LICENSE) for details.
