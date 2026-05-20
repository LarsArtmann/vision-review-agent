# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- `Analyzer` interface with `Analyze`/`AnalyzeStream` methods for consumer testability
- `ScreenshotAnalyzer` now implements `Analyzer` interface
- `MediaType` defined string type with constants (`MediaTypePNG`, `MediaTypeJPEG`, `MediaTypeGIF`, `MediaTypeWebP`)
- `NewImageSource` constructor with empty-data validation
- `ErrEmptyImageData` sentinel error for empty image data
- `ErrImageTooLarge` sentinel error for oversized images
- `io.LimitReader` in `LoadImageFromReader` with 50 MB cap to prevent OOM
- BDD test suite using Ginkgo/Gomega for user-facing behavior specs
- Compile-time interface checks for `Agent` and `ScreenshotAnalyzer`

### Changed

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

- WebP validation accepting any RIFF container (WAV, AVI) due to missing secondary magic check
- `Analyze`/`AnalyzeStream` sending `&0` for MaxOutputTokens/Temperature when not configured
- `ValidateImage` returning ad-hoc error instead of `ErrEmptyImageData`
- `AnalyzeStream` O(n^2) string concatenation for large responses
- `.gitignore` lines 31-36 missing `#` comment prefix
- `flake.nix` stale vendorHash causing `nix flake check` failure

## [0.1.0] - 2026-01-01

### Added

- Initial release
