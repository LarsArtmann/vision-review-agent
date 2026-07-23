# Vision Review Agent

## Overview

A Go SDK for building AI agents with vision capabilities. Built on top of [charm.land/fantasy](https://github.com/charmbracelet/fantasy).

## Architecture

```
cmd/vision/              CLI tool
pkg/                     Public library code
  vision/                Core SDK package
    vision.go            Agent, Config, Analyze, AnalyzeStream, AnalyzeConversation
    image.go             ImageSource, LoadImageFromFile/URL/Base64/Reader
    screenshot.go        ScreenshotAnalyzer builder (cache-safe)
    structured.go        AnalyzeStructured[T], AnalyzeStructuredStream[T]
    conversation.go      Conversation type for multi-turn analysis
    batch.go             AnalyzeBatch for concurrent analysis
    hooks.go             Hooks (OnStart, OnFinish, OnError)
    errors.go            Re-exports domain errors + ModelError classification
    validate.go          Image format validation (magic bytes)
  errors/                Centralized domain-specific errors (apperrors)
    errors.go            Sentinel validation errors
    model.go             ModelError, ErrorKind, Classify, IsRetryable
internal/                Private implementation code
  visionutil/            Internal helpers (prompt building, unmarshaling)
examples/                Working examples for each provider
```

## Key Design Decisions

- **Standalone `AnalyzeStructured[T]` / `AnalyzeStructuredStream[T]`** — Go doesn't allow type params on methods, so these are package-level functions that take a `*Agent`
- **Nil image filtering** — All analysis functions filter nil images from variadic args to prevent panics
- **Context cancellation** — `withTimeout` returns `(ctx, cancel)`; callers must `defer cancel()`
- **Validation at boundaries** — `Config.Validate()` at construction, input validation at method entry
- **Centralized errors** — Domain errors live in `pkg/errors/` and are re-exported from `pkg/vision/` for backwards compatibility
- **Classified model errors** — All model errors are wrapped in `*ModelError` with an `ErrorKind` and `IsRetryable()` for consumer retry logic
- **Table-driven tests** — Pure function tests use table-driven pattern for maintainability
- **BDD tests** — User-facing behavior tests use Ginkgo + Gomega for readability
- **Testify tests** — Error classification and feature tests use testify/require for clarity
- **Strong types** — `MediaType` is a defined string type; `ImageSource` validates at construction
- **Hooks are synchronous** — `Hooks` callbacks fire in the calling goroutine; keep them fast
- **Batch uses semaphore** — `AnalyzeBatch` bounds concurrency via `golang.org/x/sync/semaphore`
- **ScreenshotAnalyzer cache invalidation** — All `With*` builder methods set `cachedAgent = nil` to ensure config changes take effect

## Testing

```bash
just test        # Run tests with coverage
just test-race   # Run with race detector
just coverage    # Run tests and enforce 70% threshold
just vet         # Run go vet
just fmt         # Run gofmt
just lint        # Run golangci-lint
```

### Test Organization

- `*_test.go` — Table-driven tests for pure functions (config validation, image format detection)
- `*_bdd_test.go` — Ginkgo BDD specs for user-facing behavior (agent analysis, streaming, screenshot analyzer)
- `agent_suite_test.go` — Ginkgo test runner (`TestGinkgo`)
- `mock_test.go` — Shared test helpers and mock model

## Dependencies

- `charm.land/fantasy` — Core AI agent framework
- `charm.land/fantasy/providers/openai` — OpenAI provider
- `charm.land/fantasy/providers/openrouter` — OpenRouter provider (multi-model)

## Type Model

- `MediaType` — Defined string type with typed constants (`MediaTypePNG`, `MediaTypeJPEG`, etc.)
- `ImageSource` — Created via `NewImageSource(data, mediaType, filename)` which validates non-empty data
- `Analyzer` — Interface for `Analyze`/`AnalyzeStream`; consumers can mock this instead of concrete `Agent`
- `Agent` — Concrete implementation of `Analyzer`; compile-time checked via `var _ Analyzer = (*Agent)(nil)`
- `ScreenshotAnalyzer` — Fluent builder that implements `Analyzer`; delegates to cached `Agent`

- **WebP validation** — Checks both RIFF header (bytes 0-3) and WEBP magic (bytes 8-11) to reject WAV/AVI
- **Size limit** — `LoadImageFromReader` caps at 50 MB via `io.LimitReader`
- **Conditional pointers** — `Analyze`/`AnalyzeStream` only send `MaxOutputTokens`/`Temperature` when non-zero
- **strings.Builder** — `AnalyzeStream` uses builder for O(n) instead of O(n^2) string concat
- **ScreenshotAnalyzer** — Implements `Analyzer` interface via delegate methods

## Sentinel Errors

All in `pkg/errors/`, re-exported from `pkg/vision/`:

- `ErrNoModel` — No language model configured
- `ErrEmptyPrompt` — Empty prompt provided
- `ErrNoImages` — No images provided
- `ErrInvalidTemperature` — Temperature out of 0.0-2.0 range
- `ErrInvalidMaxTokens` — Negative max tokens
- `ErrInvalidTopP` — Top-p out of 0.0-1.0 range
- `ErrInvalidTopK` — Negative top-k
- `ErrInvalidPresencePenalty` — Presence penalty out of -2.0 to 2.0
- `ErrInvalidFrequencyPenalty` — Frequency penalty out of -2.0 to 2.0
- `ErrInvalidImage` — Data doesn't match known image format
- `ErrEmptyImageData` — Image data is empty
- `ErrImageTooLarge` — Image exceeds 50 MB limit

## Classified Model Errors

Model invocation errors are wrapped in `*apperrors.ModelError` (re-exported as `vision.ModelError`):

- `ErrorKind` — 11 categories: `KindRateLimited`, `KindTimeout`, `KindServerError`, `KindNetwork`, `KindAuthentication`, `KindNotFound`, `KindBadRequest`, `KindContextTooLarge`, `KindCancelled`, `KindStructuredParse`, `KindUnknown`
- `IsRetryable()` — Quick check for retry logic
- `Unwrap()` — Preserves original cause for `errors.Is` / `errors.AsType`
- Extract via `errors.AsType[*vision.ModelError](err)`

## CLI Usage

```bash
# Build
just cli

# Basic analysis
./vision-cli -prompt "Find UI bugs" screenshot.png

# JSON output
./vision-cli -json -prompt "Analyze" screenshot.png

# Streaming
./vision-cli -stream -prompt "Describe this" screenshot.png

# With timeout
./vision-cli -timeout 30s -prompt "Review" screenshot.png
```

## Code Duplication

See [`docs/DUPLICATION_POLICY.md`](docs/DUPLICATION_POLICY.md) for the full
list of accepted duplications and their rationale. Goal: **zero harmful
duplication**, not zero report lines. Idiomatic Go patterns (interface
signatures, table-driven tests, BDD assertion idioms) are kept by design.
