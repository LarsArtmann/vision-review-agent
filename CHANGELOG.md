# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- Nothing yet.

### Changed

- Nothing yet.

### Fixed

- Nothing yet.

## [0.5.1] - 2026-08-12

### Fixed

- **Critical: `.gitignore` blocked `pkg/vision/` files from being committed.**
  The pattern `vision` (without leading `/`) matched the `pkg/vision/` directory,
  silently preventing `modelinfo.go`, `modelinfo_test.go`, `cost_pricing_test.go`,
  and three `cmd/vision/` catalog-integration files from being included in the
  v0.5.0 tag. The published v0.5.0 module was uncompilable as a result
  (`undefined: ModelInfo`). Fixed by anchoring binary patterns to the repo root
  (`/vision`, `/vision-cli`, `/error-handling`). All previously-blocked files are
  now tracked.

## [0.5.0] - 2026-08-12

### Added

- **Catwalk model catalog integration** — `internal/catalog` package provides
  model discovery over [charm.land/catwalk](https://github.com/charmbracelet/catwalk),
  bringing 40+ providers and 800+ vision-capable models. The CLI now supports
  `-list-providers`, `-list-models`, `-provider-info` flags and catalog-driven
  provider construction via `BuildProvider`.
- **`pkg/vision.ModelInfo`** — public SDK type for model metadata (pricing,
  context window, capabilities). Set `Config.ModelInfo` to auto-configure
  `MaxOutputTokens` from catalog defaults and enable cost tracking.
- **`CostTracker.SetPricing` / `CostTracker.CostUSD`** — track real dollar
  costs based on catalog pricing data. `NewAgentWithCostTracker` auto-wires
  pricing from `Config.ModelInfo`.
- **Model ID suggestions** — when a model ID isn't in the catalog, the CLI
  suggests the closest match by edit distance.
- **Remote catalog sync** — set `CATWALK_URL` to enable ETag-based remote
  catalog updates with local file caching and embedded fallback.
- **Structured parse error tests** — `TestAnalyzeStructuredStreamUnmarshalFailure`
  and `TestAnalyzeStructuredUnmarshalFailure` cover the streaming and
  non-streaming unmarshal failure paths. The mock model now supports
  `streamObjectFunc` and `generateObjectResponse` fields for injecting malformed
  objects into both paths.
- **Validation error tests** — `TestValidationErrorsIncludeOffendingValues`
  (7 table cases) verifies that every ranged sentinel in `Config.Validate`
  includes the offending value in the message while preserving `errors.Is`
  matching. `TestNoModelReturnsBareSentinel` confirms the model-missing path
  returns an unwrapped sentinel.
- **Enriched error-handling example** — `examples/error-handling/main.go` now
  demonstrates both error categories: config validation sentinels (via
  `errors.Is`) and classified model errors (via `errors.AsType`).

### Changed

- **Config.Validate enriched** — all 6 ranged validation sentinels now wrap
  with the offending value: `fmt.Errorf("%w: got %v, want ...", sentinel, val)`.
  `errors.Is` still matches the sentinel; the message is self-diagnosing (e.g.
  `"vision agent: temperature must be between 0.0 and 2.0: got 3.50"`).
- **LoadImageFromURL context** — bare `return nil, err` paths now wrap with the
  URL: `fmt.Errorf("download image from %q: %w", url, err)`.
- **Agent construction context** — `NewAgentWithCostTracker` and
  `ScreenshotAnalyzer.agent()` failures now include operation context instead of
  bare propagation.

### Fixed

- **Provider alias in ModelInfo lookup** — `FindModelInProvider` now normalizes
  the provider name (`google` → `gemini`) before catalog lookup, so ModelInfo
  is correctly populated for the Google Gemini provider.
- **Remote sync timeout** — `CATWALK_URL` sync now uses a 5-second context
  timeout instead of the catwalk client default (30s), preventing long startup
  delays when the remote server is unreachable.
- **Duplicate mock model** — `integrationMockModel` removed; integration tests
  now reuse the existing `cliMockModel`.
- **Usage hints** — CLI help text now lists specific provider env var names
  (OPENAI_API_KEY, ANTHROPIC_API_KEY, etc.) instead of a generic message.
- **Silent unmarshal swallow in AnalyzeStructuredStream** — the final-object
  unmarshal error in `ObjectStreamPartTypeFinish` was discarded with
  `_ = visionutil.UnmarshalToType(...)`. It now returns a `KindStructuredParse`
  `*ModelError` so callers can detect and handle parse failures.
- **BDD test assertions** — `gomega.Equal(sentinel)` updated to
  `gomega.MatchError(sentinel)` + `ContainSubstring` to correctly match wrapped
  sentinels via `errors.Is`.

> **Known issues carried forward (not yet fixed):**
>
> - **Tag anomaly (partial).** `v0.2.1` (below) still points to commit
>   `d5dda4b`, a pre-`[0.2.0]` ancestor, and does not represent a real release.
>   The `v0.3.0` ghost was deleted in `[0.4.0]` (see that section). Deleting
>   `v0.2.1` too is deferred — destructive remote-tag op with no functional
>   impact (it sorts below `[0.2.0]`/`[0.4.0]`).

## [0.4.0] - 2026-07-28

> **Release note — tag cleanup.** The `v0.3.0` tag was a ghost: it pointed at
> commit `d5dda4b` ("Improve test formatting and readability across vision
> package", dated 2026-04-27), an ancestor of `[0.2.0]` (`003a256`, 2026-07-23),
> and never represented a real release. As part of this release the bogus
> `v0.3.0` tag was deleted (local + remote) and the real post-v0.2.0 body of
> work ships here. It ships as `[0.4.0]` rather than a fresh `v0.3.0` because
> `v0.3.0` is permanently burned on `proxy.golang.org` as `d5dda4b`; reusing
> the number would cause checksum mismatches for any consumer who already
> resolved the ghost. `v0.2.1` remains a ghost on the same commit for now (see
> "Known issues" above).

### Added

- **Config.Retry** — optional `*RetryConfig` field on `Config` that enables
  vision-layer automatic retry of transient failures across all non-streaming
  analysis methods (Analyze, AnalyzeConversation, AnalyzeStructured,
  AnalyzeBatch). Composes with `MaxRetries` (fantasy HTTP-layer retry) for
  layered retry. Streaming methods do not auto-retry (ambiguous delta
  semantics).
- **Config.Preprocess** — optional `*PreprocessConfig` field on `Config` that
  auto-resizes images before every `Analyze*` call. `PreprocessImage` function
  and `ScreenshotAnalyzer.WithMaxDimension` builder also added. `JPEGQuality`
  is now wired through the full resize/encode path.
- **Image compression** — `CompressImage(img, quality)` re-encodes JPEGs at a
  lower quality without resizing (PNG input preserves its format).
- **NewAgentWithCostTracker** — convenience constructor that auto-wires a
  `CostTracker` into `Hooks.OnFinish`, composing with any user-supplied hooks.
- **New ErrorKinds** — `KindNotImplemented` (HTTP 501, not retryable),
  `KindServiceUnavailable` (HTTP 503, retryable), `KindContentFilter` (content
  policy rejection from 400 with signal-phrase detection, not retryable).
- **CLI tests** — `cmd/vision/main_test.go` covering `adviceForKind`,
  `buildConfig`, `parseTimeout`, `createProvider` error paths, and a full
  `parseFlags` table (defaults, all flags, `-version`, missing args, unknown
  flag). `parseFlags` was refactored to accept a `*flag.FlagSet` and return
  errors instead of calling `os.Exit`, enabling isolated testing.
- **Coverage for new code** — `mediaTypeFromExtension` table test, BMP
  decode→resize roundtrip, `PreprocessImage` passthrough, `Config.Preprocess`
  end-to-end wiring in `Analyze`/`AnalyzeStructured`, `NewAgentWithCostTracker`
  nil-`RawResponse` contract, `contentFilter` signal detection, 501/503 via full
  `Analyze` path, and a genuine `AnalyzeBatch` mixed success+error test.
- **BDD error-classification specs** — `error_classification_bdd_test.go` with
  10 table entries covering consumer retry decisions via Ginkgo.
- **Batch classified-error tests** — `AnalyzeBatch` per-image error
  classification verified.
- **`examples/error-handling/main.go`** — consumer-facing example showing
  `errors.AsType[*ModelError]` → kind-lookup pattern.
- **CI workflow** — `.github/workflows/ci.yml` with build, vet, race test,
  coverage gate (≥70%), lint, format check, **plus** a `go mod tidy` diff
  check, a `golangci-lint config verify` step, and a dedicated
  `nix-flake-check` job.
- **Hooks across all analysis methods** — `OnStart`/`OnFinish`/`OnError` now
  fire in `AnalyzeConversation`, `AnalyzeConversationStream`,
  `AnalyzeStructured`, and `AnalyzeStructuredStream` (previously only
  `Analyze`/`AnalyzeStream`). Structured methods use a synthesized
  `*AnalyzeResult` — see `AnalyzeResult.RawResponse` doc for the nil contract.
- **Retry middleware** — `WithRetry[T]` generic function with `RetryConfig`
  (exponential backoff + jitter, honors `IsRetryable()`). Zero-value
  `RetryConfig` falls back to `DefaultRetryConfig()` (3 attempts).
- **Cost tracking** — `CostTracker` thread-safe token accumulator;
  `NewCostTracker()`, `Add`, `AddResult`, `Total`, `Calls`. Integrates with
  `Hooks.OnFinish` via `AddResult`.
- **Image preprocessing** — `ResizeImage` Catmull-Rom downscale
  (aspect-preserving, longest-side cap); decodes PNG/JPEG/WebP/GIF.
- **Custom HTTP client** — `LoadImageFromURLWithClient` variant of
  `LoadImageFromURL` accepting a `*http.Client` (proxies, timeouts, TLS).
- **`MediaTypeBMP`** constant added to the `MediaType` enum.
- **`Conversation.Clear`** — resets message history, returns the same instance.
- **`ScreenshotAnalyzer.WithHooks`** — fluent builder method with cache
  invalidation.
- **Advanced config capabilities (fantasy passthrough):** `Config.Tools`,
  `Config.ToolChoice`, `Config.StopConditions`, `Config.PrepareStep`,
  `Config.Headers`, `Config.UserAgent`.
- **CLI providers** — Anthropic (`ANTHROPIC_API_KEY`), Google (Application
  Default Credentials), `openaicompat` (`OPENAICOMPAT_BASE_URL`) added to the
  CLI (build-verified; no runtime credentials tested).
- **CLI `-structured` flag** — built-in `uiReview` schema producing structured
  JSON output.
- **Fuzz tests** — `FuzzDetectImageFormat`, `FuzzDecodeBase64Flex`.
- **Examples** — `examples/conversation`, `examples/batch`, `examples/hooks`,
  `examples/structured-stream`, `examples/url-loading`.
- `.editorconfig` for consistent formatting across editors.

### Changed

- **Retry tests are fast and deterministic** — removed `MaxRetries: 1` from the
  vision-layer retry tests (it was enabling fantasy's ~5s HTTP backoff on every
  retryable mock call). Vision-layer retry is now exercised in isolation with
  exact call-count assertions. Full race suite dropped from ~11s to ~3.6s.
- **Shared encode path** — `ResizeImage`/`ResizeImageWithQuality`/`CompressImage`
  now share a single `encodeImage` helper; PNG output uses `BestCompression`.
- **License metadata corrected** — `flake.nix` now reads `licenses.unfree`
  matching the PROPRIETARY `LICENSE` file. The `[0.2.0]` false claim is
  resolved.
- **BMP decoder registered** — `golang.org/x/image/bmp` blank import added to
  `preprocess.go`, so `image.Decode` can decode BMP. Previously BMP was
  detected by magic bytes but could not be decoded for resize.
- **`mediaTypeFromExtension` now has explicit known-extension table** — PNG,
  JPEG, GIF, WebP, BMP are matched deterministically before falling back to
  system-dependent `mime.TypeByExtension`. A `.bmp` file is no longer
  mislabeled as PNG.
- **Model parameter duplication eliminated** — `applyModelParamsAgentCall` and
  `applyModelParamsStreamCall` replaced by a single `Config.optionalParams()`
  helper used across all 4 call sites (vision.go + structured.go).
- **golangci-lint config: `nolintlint` tightened** with `require-explanation:
true`; all `//nolint:` directives now carry explanations. `funlen` config
  fixed for v2 schema. `golangci-lint config verify` passes.
- `AnalyzeConversationStream` now uses the shared `validateAnalyzeInput`
  helper (was inline `prompt == ""` + `requireImages` checks).
- AGENTS.md rewritten: all deprecated `just` command references replaced with
  flake equivalents; `GOWORK=off` requirement documented.
- golangci-lint config: depguard `$module` variable replaced with the
  hardcoded module path (works around a golangci-lint v2 regression);
  6 dead `//nolint:legacyerrors` directives removed; project-wide lint issues
  driven from 39 to 0.

### Removed

- **`applyModelParamsAgentCall` / `applyModelParamsStreamCall`** — replaced by
  `Config.optionalParams()`.
- **Dead `errTestNoop` sentinel** — removed from `pkg/errors/model_test.go`.
- **`wrapNoop`** — replaced by `wrapChain` (real `fmt.Errorf("%w")` wrapper)
  that actually tests error-chain traversal through classification.
- **`wrapWithPrompt`** function — deleted from `vision.go`; `ScreenshotAnalyzer`
  call sites now return config-validation errors directly (model errors are
  already classified via delegation).
- **`applyOptionalPointers`** helper — replaced with direct field assignment in
  `applyModelParamsAgentCall` / `applyModelParamsStreamCall`.

### Fixed

- **Broken `examples/error-handling/main.go`** — `handleError` called `log.Fatalf`
  on every path (even successful advice printing) and had a dead `!found` branch.
  Rewritten as `printModelError` that prints advice to stderr and lets the caller
  decide the exit code. No more always-exit-1 consumer example.
- **`isContentFilterRejection` false positive** — the bare `"safety"` signal
  matched benign provider messages (e.g. "safety-related best practice"). Replaced
  with specific phrases (`"safety filter"`, `"safety policy"`, `"blocked for
safety"`, `"removed for safety"`) that only match real content-policy rejections.
- **Stale `version` constant** — `cmd/vision/main.go` hardcoded `"0.2.0"` despite
  `[Unreleased]` features. Changed to `var version = "0.3.0-dev"` (honest for
  unreleased) and wired `-ldflags "-X main.version=..."` in `flake.nix` so tagged
  builds inject the real semver.
- **Lint gate fully clean** — fixed `testifylint: float-compare` (3×
  `require.InDelta` → `require.InEpsilon`), `internal/cli/helpers.go`
  `nlreturn`+`wsl_v5`, `examples/openai` `golines`, and all stale
  `examples/error-handling` formatting warnings.
- **`CompressImage` edge case** — now returns the original image unchanged when
  re-encoding would not shrink it (e.g. compressing an already-low-quality JPEG
  at a higher quality), instead of silently inflating it.
- `LoadImageFromURL` now runs `ValidateImage` after download, rejecting non-image
  response bodies.
- ScreenshotAnalyzer cache invalidation: all `With*` builder methods now set
  `cachedAgent = nil`.

### Changed

- **cmd/vision coverage 37.9% → 73.3%** — added tests for `loadImages`,
  `printJSON`, `printText`, `runAnalysis` (text/json/stream/structured branches),
  `runStructured`, `createProvider` (openaicompat happy + missing-baseURL),
  and `printAnalysisError` (classified + unclassified).
- **`infertypeargs` cleanup** — removed unnecessary explicit `[testReview]` type
  arguments from `AnalyzeStructuredStream` calls where Go can infer T from the
  callback; kept them where `nil` callbacks prevent inference.
- **DOMAIN_LANGUAGE.md updated** — added `CompressImage`, `ResizeImageWithQuality`,
  `encodeImage`, `parseFlags`, two-layer retry architecture note, and CLI context.
- **README snippets compile-verified** — all 13 Go code blocks extracted and
  built against the real module; every API call, type, and method signature is
  correct.

## [0.3.0] - 2026-07-27

> **Tag anomaly (resolved in `[0.4.0]`).** This tag pointed to commit
> `d5dda4b` ("Improve test formatting and readability across vision package",
> dated 2026-04-27), an **ancestor** of `[0.2.0]`. It never represented a real
> release. The tag was **deleted** in `[0.4.0]`; the real post-v0.2.0 work ships
> there. Kept here as a historical record — the version number is permanently
> burned on `proxy.golang.org` as `d5dda4b`.

### Changed

- Test formatting and readability improvements across the vision package (same
  commit as `[0.2.1]`).

## [0.2.1] - 2026-07-27

> **Tag anomaly:** this tag points to the **same commit** as `[0.3.0]`
> (`d5dda4b`). See the note under `[0.3.0]`.

### Changed

- Test formatting and readability improvements across the vision package.

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
  > **Correction (retroactive, `[Unreleased]`):** the v0.2.0 tag shipped
  > `flake.nix` with `licenses.mit`, so this line was inaccurate as published.
  > The actual change to `licenses.unfree` landed in `[Unreleased]`. Recorded
  > here for honesty, not to rewrite history.
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
