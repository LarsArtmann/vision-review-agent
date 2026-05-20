# Vision Review Agent

A simple, production-ready Go SDK for building AI agents with vision capabilities. Built on top of [charm.land/fantasy](https://github.com/charmbracelet/fantasy).

## Features

- **Simple API** — Analyze images/screenshots with a single function call
- **Multi-provider** — Works with OpenAI, OpenRouter, and any fantasy-compatible provider
- **Streaming** — Stream responses in real-time
- **Structured Output** — Get typed, structured results instead of free-form text
- **Built-in CLI** — Analyze images from the command line
- **Validation** — Strong input validation with clear error types
- **Configurable** — Temperature, token limits, retries, timeouts

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
    WithMaxOutputTokens(1000)

result, _ := analyzer.AnalyzeScreenshot(ctx, "Review this page", "screenshot.png")
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
