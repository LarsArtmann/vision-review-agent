# Vision Review Agent

**Version:** 0.1.0 | **Updated:** May 20, 2026

## Overview

A Go SDK for building AI agents with vision capabilities. Built on top of [charm.land/fantasy](https://github.com/charmbracelet/fantasy).

## Architecture

```
cmd/vision/              CLI tool
pkg/                     Public library code
  vision/                Core SDK package
    vision.go            Agent, Config, AnalyzeResult
    image.go             ImageSource, loading helpers
    screenshot.go        ScreenshotAnalyzer builder
    structured.go        Typed structured output (AnalyzeStructured[T])
    errors.go            Re-exports domain errors (backwards compat)
    validate.go          Image format validation (magic bytes)
  errors/                Centralized domain-specific errors (apperrors)
internal/                Private implementation code
  visionutil/            Internal helpers (prompt building, unmarshaling)
examples/                Working examples for each provider
```

## Key Design Decisions

- **Standalone `AnalyzeStructured[T]`** — Go doesn't allow type params on methods, so it's a package-level function that takes a `*Agent`
- **Nil image filtering** — All analysis functions filter nil images from variadic args to prevent panics
- **Context cancellation** — `withTimeout` returns `(ctx, cancel)`; callers must `defer cancel()`
- **Validation at boundaries** — `Config.Validate()` at construction, input validation at method entry
- **Centralized errors** — Domain errors live in `pkg/errors/` and are re-exported from `pkg/vision/` for backwards compatibility
- **Table-driven tests** — Pure function tests use table-driven pattern for maintainability
- **BDD tests** — User-facing behavior tests use Ginkgo + Gomega for readability
- **Strong types** — `MediaType` is a defined string type; `ImageSource` validates at construction

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

## Image Validation

Uses magic byte signatures to validate image formats:

- PNG: `89 50 4E 47`
- JPEG: `FF D8 FF`
- GIF: `47 49 46`
- WebP: `52 49 46 46` (RIFF)
- BMP: `42 4D`

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
