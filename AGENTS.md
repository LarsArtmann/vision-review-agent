# Vision Review Agent

**Version:** 0.1.0 | **Updated:** April 27, 2026

## Overview

A Go SDK for building AI agents with vision capabilities. Built on top of [charm.land/fantasy](https://github.com/charmbracelet/fantasy).

## Architecture

```
cmd/vision/         CLI tool
vision/             Core SDK package
  vision.go         VisionAgent, Config, AnalyzeResult
  image.go          ImageSource, loading helpers
  screenshot.go     ScreenshotAnalyzer builder
  structured.go     Typed structured output (AnalyzeStructured[T])
  errors.go         Sentinel error types
  validate.go       Image format validation (magic bytes)
examples/           Working examples for each provider
```

## Key Design Decisions

- **Standalone `AnalyzeStructured[T]`** — Go doesn't allow type params on methods, so it's a package-level function that takes a `*VisionAgent`
- **Nil image filtering** — All analysis functions filter nil images from variadic args to prevent panics
- **Context cancellation** — `withTimeout` returns `(ctx, cancel)`; callers must `defer cancel()`
- **Validation at boundaries** — `Config.Validate()` at construction, input validation at method entry

## Testing

```bash
make test        # Run tests
make test-race   # Run with race detector
make vet         # Run go vet
make fmt         # Run gofmt
```

## Dependencies

- `charm.land/fantasy` — Core AI agent framework
- `charm.land/fantasy/providers/openai` — OpenAI provider
- `charm.land/fantasy/providers/openrouter` — OpenRouter provider (multi-model)

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
make cli

# Basic analysis
./vision-cli -prompt "Find UI bugs" screenshot.png

# JSON output
./vision-cli -json -prompt "Analyze" screenshot.png

# Streaming
./vision-cli -stream -prompt "Describe this" screenshot.png

# With timeout
./vision-cli -timeout 30s -prompt "Review" screenshot.png
```
