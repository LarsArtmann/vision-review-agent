# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

## [0.2.0] - 2026-07-23

### Added

- **Structured output:** package-level generic functions `AnalyzeStructured[T]` and `AnalyzeStructuredStream[T]` that generate a JSON schema from `T` and return a typed `*fantasy.ObjectResult[T]`
- **Multi-turn conversation:** `Conversation` type (`NewConversation`, `AddUserMessage`, `AddAssistantMessage`, `Messages`, `Len`) plus `AnalyzeConversation`/`AnalyzeConversationStream` methods for follow-up questions with history
- **Batch analysis:** `AnalyzeBatch` analyzes many images concurrently with bounded concurrency (semaphore), per-image error capture, and ordered results
- **Lifecycle hooks:** `Hooks` struct with `OnStart`, `OnFinish`, and `OnError` callbacks for logging/metrics; nil-safe and synchronous
- **Classified model errors:** centralized `pkg/errors` package with `ModelError`, 11 `ErrorKind` categories, `IsRetryable()`, `Unwrap()`, `Wrap`, and `Classify` — re-exported from `pkg/vision` as `vision.ModelError`, `vision.Classify`, and `vision.IsRetryable`
- **Screenshot analyzer:** fluent `ScreenshotAnalyzer` builder with `With*` methods and `AnalyzeScreenshot`/`AnalyzeScreenshots`/`AnalyzeScreenshotImages`/`AnalyzeConversation` convenience methods; all builders invalidate the cached agent
- **CLI tool:** `cmd/vision` with flags for provider, model, prompt, system prompt, streaming, temperature, max tokens, timeout, JSON output, and version; prints actionable advice for classified model errors
- **Image loading:** `LoadImageFromURL`, `LoadImageFromBase64`, `LoadImageFromReader`, and `NewImageSource` constructors alongside `LoadImageFromFile`
- **Image validation:** `ValidateImage`, `IsValidImage`, and `DetectImageFormat` via magic-byte signatures (PNG, JPEG, GIF, WebP)
- **Extra config fields:** `TopP`, `TopK`, `PresencePenalty`, `FrequencyPenalty`, `MaxRetries`, `RequestTimeout`, and `Hooks` on `Config`
- `Analyzer` interface with `Analyze`/`AnalyzeStream` methods for consumer testability
- `ScreenshotAnalyzer` now implements the `Analyzer` interface
- `MediaType` defined string type with constants (`MediaTypePNG`, `MediaTypeJPEG`, `MediaTypeGIF`, `MediaTypeWebP`)
- `NewImageSource` constructor with empty-data validation
- `ErrEmptyImageData` sentinel error for empty image data
- `ErrImageTooLarge` sentinel error for oversized images
- `io.LimitReader` in `LoadImageFromReader` with 50 MB cap to prevent OOM
- BDD test suite using Ginkgo/Gomega for user-facing behavior specs
- Compile-time interface checks for `Agent` and `ScreenshotAnalyzer`

### Changed

- CLI `-json` output now uses camelCase keys (`inputTokens`, `outputTokens`, `totalTokens`)
- CLI provider/env-var errors now use wrapped sentinel errors (inspectable via `errors.Is`)
- golangci-lint config scoped `forbidigo`, `ireturn`, and `mnd` to library code only; application paths (`cmd/`, `examples/`, `internal/cli/`) are excluded where the pattern is intentional
- `MediaType` changed from `string` constants to defined type `type MediaType string`
- `LoadImageFromReader` signature changed from `string` to `MediaType` for mediaType param
- `AnalyzeStream` now uses `strings.Builder` instead of string concatenation (O(n) vs O(n^2))
- `Analyze`/`AnalyzeStream` only send `MaxOutputTokens` and `Temperature` pointers when explicitly configured (non-zero)
- `ValidateImage` now uses `ErrEmptyImageData` sentinel instead of ad-hoc `errors.New`
- WebP validation now checks secondary WEBP magic at offset 8 (rejects WAV/AVI)
- Centralized errors in `pkg/errors/`, re-exported from `pkg/vision/` for backward compat
- `flake.nix` vendorHash updated to match current dependencies
- License metadata corrected to `unfree` (matching PROPRIETARY LICENSE file)
- README updated: removed non-existent Anthropic provider reference, added new error types, corrected license

### Removed

- Duplicate table-driven tests replaced by BDD suite (`screenshot_test.go`, `image_test.go`)
- Unused `AssertEq` and `AssertError` test helpers from `mock_test.go`

### Fixed

- Resolved all pre-existing golangci-lint failures so the release passes `golangci-lint run ./...` cleanly
- CLI image buffer built with `make(..., 0, n)` + `append` instead of a non-zero-length slice (makezero)
- WebP validation accepting any RIFF container (WAV, AVI) due to missing secondary magic check
- `Analyze`/`AnalyzeStream` sending `&0` for MaxOutputTokens/Temperature when not configured
- `ValidateImage` returning ad-hoc error instead of `ErrEmptyImageData`
- `AnalyzeStream` O(n^2) string concatenation for large responses
- `.gitignore` lines 31-36 missing `#` comment prefix
- `flake.nix` stale vendorHash causing `nix flake check` failure

## [0.1.0] - 2026-01-01

### Added

- Initial release
