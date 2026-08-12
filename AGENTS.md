# Vision Review Agent

## Overview

A Go SDK for building AI agents with vision capabilities. Built on top of [charm.land/fantasy](https://github.com/charmbracelet/fantasy).

## Architecture

```
cmd/vision/              CLI tool
  main.go                Catalog-driven provider construction, listing flags, analysis
  listing.go             printProviders, printVisionModels, printProviderInfo, suggestModel
pkg/                     Public library code
  vision/                Core SDK package
    vision.go            Agent, Config, Analyze, AnalyzeStream, AnalyzeConversation
    modelinfo.go         ModelInfo type, NewModelInfo(catwalk.Model), applyModelInfoDefaults
    image.go             ImageSource, LoadImageFromFile/URL/Base64/Reader
    screenshot.go        ScreenshotAnalyzer builder (cache-safe)
    structured.go        AnalyzeStructured[T], AnalyzeStructuredStream[T]
    conversation.go      Conversation type for multi-turn analysis
    batch.go             AnalyzeBatch for concurrent analysis
    cost.go              CostTracker (with SetPricing + CostUSD), NewAgentWithCostTracker
    hooks.go             Hooks (OnStart, OnFinish, OnError)
    errors.go            Re-exports domain errors + ModelError classification
    validate.go          Image format validation (magic bytes)
  errors/                Centralized domain-specific errors (apperrors)
    errors.go            Sentinel validation errors
    model.go             ModelError, ErrorKind, Classify, IsRetryable
internal/                Private implementation code
  catalog/               Model catalog access layer (over charm.land/catwalk)
    catalog.go           Service type, FindProvider, FindModel, VisionModels
    provider.go          BuildProvider: catwalk.Type → fantasy constructor bridge
    sync.go              Remote sync with ETag caching + file cache management
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
- **Hooks fire in every analysis path** — Analyze, AnalyzeStream, AnalyzeConversation(Stream), AnalyzeStructured(Stream) all call fireStart/fireFinish/fireError after validation
- **Structured hooks RawResponse is nil** — `AnalyzeResult.RawResponse` is nil for synthesized results from AnalyzeStructured/AnalyzeStructuredStream (those return `*fantasy.ObjectResult`, not `*fantasy.AgentResult`). See the field doc.
- **WithRetry[T] is external middleware** — Retries are opt-in via a generic wrapper (not baked into Agent) so callers control policy; respects `IsRetryable()`
- **Config.Retry layers vision-level retry** — When set, retries the whole model invocation with `RetryConfig` backoff+jitter; composes with `MaxRetries` (fantasy HTTP-layer retry). Streaming methods do NOT auto-retry.
- **MaxRetries defaults to 0 (no HTTP-layer retry)** — `NewAgent` always forwards `MaxRetries` to fantasy; the zero value disables retries entirely (fantasy's `MaxRetries==0 → no retry`). Previously zero meant "fantasy default" (3 retries, 5+10+20s backoff), which caused 35s test stalls and OOM under `-race`. Set `MaxRetries` explicitly or use `Config.Retry` for retry behavior.
- **Config.Preprocess auto-applies** — When set, `PreprocessConfig.MaxDimension` resizes images before every `Analyze*` call. `ScreenshotAnalyzer.WithMaxDimension` sets it fluently.
- **CostTracker integrates via Hooks.OnFinish** — Thread-safe; `NewAgentWithCostTracker` auto-wires it
- **NewAgentWithCostTracker** — Convenience constructor that composes cost tracking with user hooks
- **Capability fields are passthrough** — `Config.Tools/ToolChoice/StopConditions/PrepareStep/Headers/UserAgent` map 1:1 to fantasy options; nil/empty means provider default
- **optionalParams() eliminates duplication** — Single `Config.optionalParams()` helper feeds model params to all 4 call sites (AgentCall, AgentStreamCall, ObjectCall x2)
- **withPrepared[T] owns the analysis prologue/epilogue** — Generic higher-order wrapper (`pkg/vision/vision.go`) that runs `prepare()` (validate → preprocess → fireStart → timeout → toFileParts), defers `cancel()`, then invokes a closure with the prepared request. All 6 analysis methods (4 `Analyze*` + 2 `AnalyzeStructured*`) use it, so the `if err != nil { return } + defer cancel()` idiom appears once. Free function (not a method) because Go disallows generic methods.
- **ScreenshotAnalyzer.invalidate() centralizes cache busting** — All `With*` builder methods call `sa.invalidate()` (sets `cachedAgent = nil`) instead of inlining the assignment
- **BMP fully supported** — Decoder registered via `golang.org/x/image/bmp` blank import; `mediaTypeFromExtension` has explicit table for PNG/JPEG/GIF/WebP/BMP
- **ResizeImage returns the same instance when no resize is needed** — Avoids unnecessary re-encoding
- **Quality wired end-to-end** — `PreprocessConfig.JPEGQuality` flows through `ResizeImageWithQuality` → shared `encodeImage` helper. `ResizeImage` is a thin wrapper using the default quality (85). `CompressImage` re-encodes JPEGs without resizing (PNG preserves format via `png.BestCompression`). `encodeImage` is the single encode path shared by resize + compress
- **LoadImageFromURL validates magic bytes** — Rejects non-image HTTP bodies via `ValidateImage`
- **`isContentFilterRejection` uses specific signal phrases** — not bare words like "safety" (which matched benign messages); requires the rejection mechanism ("filter", "policy", "blocked", "removed") alongside the topic
- **`CompressImage` no-ops when output wouldn't shrink** — returns the original image if re-encoding produces equal-or-larger bytes; contract is size reduction, not format normalization
- **`version` is a `var` (not `const`)** — default `"0.3.0-dev"` for honest unreleased state; `flake.nix` injects the real semver via `-ldflags "-X main.version=..."`
- **CLI parseFlags is testable** — `parseFlags(fs *flag.FlagSet, args []string) (*config, error)` takes a FlagSet and returns errors instead of calling `os.Exit`. `main()` passes `flag.CommandLine`; tests pass a fresh `flag.ContinueOnError` set with `io.Discard` output. Version/no-args decisions surface as `cfg.showVersion` / `cfg.args` so the caller acts on them.
- **Retry tests must NOT set MaxRetries** — Vision-layer retry tests leave `MaxRetries` at 0 (default) and rely solely on `Config.Retry`. Setting `MaxRetries: 1` re-enables fantasy's HTTP-layer retry (~5s backoff per retryable mock call) and makes call counts non-deterministic. The full race suite is ~3.6s.
- **Dual json v1+v2 support — do NOT migrate imports** — All code imports only `encoding/json` (the v1 path). This transparently supports BOTH regimes: default Go (v1 behavior) AND `GOEXPERIMENT=jsonv2` (v2 behavior), because the jsonv2 experiment swaps the _implementation_ of `encoding/json` while preserving the v1 API surface (`Marshal`, `Unmarshal`, `NewEncoder`, `SetIndent`, `Decoder`). The auto-upgrade daemon repeatedly tried to switch imports to `encoding/json/v2` and `encoding/json/jsontext` — those paths are NOT supported here: they require a `go.mod` replace directive AND have a different low-level API (`jsontext.Encoder` has no `SetIndent`), which broke compilation. CI runs a dedicated `jsonv2-compat` job (`GOEXPERIMENT=jsonv2 go build/vet/test`) to guard this. Verified passing under both Go 1.26.5 default and jsonv2.
- **Validation errors include offending values** — `Config.Validate()` wraps each sentinel with `fmt.Errorf("%w: got %v, want ...", sentinel, value)`. This preserves `errors.Is` matching while making the error self-diagnosing: `"vision agent: temperature must be between 0.0 and 2.0: got 3.50, want [0.0, 2.0]"`. Tests use `require.ErrorIs` (which traverses wraps) and `require.Contains` (which checks the message).
- **No context dumping in error messages** — Variables that are RESULTS of a failed operation (`decoded`, `data`, `img`, `jsonBytes`) are never included in error messages. They are nil/garbage on the error path. Only INPUTS relevant to diagnosis (path, url, mediaType, filename, offending value) are included. The `erraudit` tool's `context_loss` findings on result variables are false positives — adding them would produce misleading error strings.
- **Bare `return err` is intentional at boundary sentinels** — When a function returns a sentinel error (`ErrEmptyPrompt`, `ErrNoImages`, `ErrInvalidImage`), propagating it without wrapping is correct: the sentinel IS the error. Wrapping `"analyze: %w"` would just add noise. Context wrapping is reserved for sites where the calling context genuinely adds diagnostic value (URL, operation name, image index).
- **Structured streaming final object unmarshal is a hard error** — `AnalyzeStructuredStream`'s `ObjectStreamPartTypeFinish` case now returns a `KindStructuredParse` error if the final object fails to unmarshal, instead of silently swallowing it and returning a zero-value T. Partial object unmarshal failures during streaming are still tolerated (best-effort).
- **`errors.AsType[E]` for typed extraction, `errors.Is` for sentinels** — The codebase uses the Go 1.26 generic `errors.AsType[*T](err)` for extracting `*ModelError`, `*fantasy.ProviderError`, `*fantasy.NoObjectGeneratedError`. It uses `errors.Is` for stdlib sentinels (`context.Canceled`, `context.DeadlineExceeded`). No legacy `errors.As` remains.
- **Functions return `error` interface, not concrete types** — Every function returns `error`, not a specific error type. This is idiomatic Go. The `erraudit` `generic_return` warnings suggesting per-function error types are false positives that would break composability and add massive boilerplate.
- **Catwalk is a metadata catalog, not a provider factory** — `internal/catalog` wraps `charm.land/catwalk` for model discovery (40+ providers, 800+ vision models). Provider construction still uses fantasy constructors via `BuildProvider`, which switches on `catwalk.Type` to call the right fantasy `New()`.
- **Embedded-first catalog** — By default, `catalog.New()` returns `embedded.GetAll()` (compiled into binary, no network). Set `CATWALK_URL` to enable remote sync with ETag caching; falls back to cached then embedded on any error.
- **`errEnvVarNotSet` aliases `catalog.ErrAPIKeyNotSet`** — The CLI's error var is set to `catalog.ErrAPIKeyNotSet` so existing `errors.Is` tests match without changes.
- **`normalizeProviderName("google") → "gemini"`** — Catwalk uses "gemini" as the InferenceProvider ID for Google; the CLI aliases "google" for backward compatibility.
- **`openaicompat` is a CLI-only fallback** — Not in the catwalk catalog. When `-provider openaicompat` is passed, the CLI reads `OPENAICOMPAT_BASE_URL`/`OPENAICOMPAT_API_KEY` directly.
- **`ModelInfo` is optional and backward compatible** — `Config.ModelInfo *ModelInfo` is nil-safe. When set, `applyModelInfoDefaults` fills in `MaxOutputTokens` from catalog defaults if unset. `NewAgentWithCostTracker` auto-wires pricing from `ModelInfo` via `CostTracker.SetPricing`.
- **`CostTracker.CostUSD()` returns 0 without pricing** — No breaking change: existing `CostTracker` users see no cost unless `SetPricing` is called or `ModelInfo` is set.

## Build & Test Commands

There is **no justfile/makefile** in this repo. Use `go` directly or the nix flake.

```bash
# Go toolchain (preferred for fast iteration)
go build ./...              # Build everything
go test ./...               # Run all tests
go test -race ./...         # Run with race detector
go vet ./...                # Run go vet
gofmt -l .                  # Check formatting
golangci-lint run ./...     # Lint (matches flake `lint` app)

# Nix flake (reproducible)
nix run .#test              # go test -race -v -coverprofile=coverage.out ./...
nix run .#lint              # golangci-lint run ./...
nix build .                 # Build the package
```

### GOWORK

This repo has **no `go.work`** — it is a single module. `go build`/`go test`
work without any special env. The nix devShell sets `GOWORK=off` defensively:
if a **parent** directory (e.g. `~/projects/go.work`) defines a workspace that
pulls this repo in, `GOWORK=off` isolates the build. Outside the nix shell, set
`GOWORK=off` only if you hit `go.work file ... not found` or module-resolution
errors pointing at a sibling module.

### Test Organization

- `*_test.go` — Table-driven tests for pure functions (config validation, image format detection, retry, cost tracking, CLI advice)
- `*_bdd_test.go` — Ginkgo BDD specs for user-facing behavior (agent analysis, streaming, screenshot analyzer, error classification)
- `agent_suite_test.go` — Ginkgo test runner (`TestGinkgo`)
- `mock_test.go` — Shared test helpers and mock model (supports retry sequences via `generateErrs`)
- `cmd/vision/main_test.go` — CLI tests (advice mapping, config building, provider error paths)

## Dependencies

- `charm.land/fantasy` — Core AI agent framework
- `charm.land/fantasy/providers/openai` — OpenAI provider
- `charm.land/fantasy/providers/openrouter` — OpenRouter provider (multi-model)
- `charm.land/catwalk` — Model catalog (40+ providers, 800+ vision models, pricing, capabilities)

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
- `ErrInvalidTemperature` — Temperature out of 0.0-2.0 range (wrapped: includes offending value)
- `ErrInvalidMaxTokens` — Negative max tokens (wrapped: includes offending value)
- `ErrInvalidTopP` — Top-p out of 0.0-1.0 range (wrapped: includes offending value)
- `ErrInvalidTopK` — Negative top-k (wrapped: includes offending value)
- `ErrInvalidPresencePenalty` — Presence penalty out of -2.0 to 2.0 (wrapped: includes offending value)
- `ErrInvalidFrequencyPenalty` — Frequency penalty out of -2.0 to 2.0 (wrapped: includes offending value)
- `ErrInvalidImage` — Data doesn't match known image format
- `ErrEmptyImageData` — Image data is empty
- `ErrImageTooLarge` — Image exceeds 50 MB limit

Wrapped sentinels use `fmt.Errorf("%w: got %v, want ...", sentinel, value)`, so `errors.Is`
still matches. Example: `"vision agent: temperature must be between 0.0 and 2.0: got 3.50, want [0.0, 2.0]"`.

## Classified Model Errors

Model invocation errors are wrapped in `*apperrors.ModelError` (re-exported as `vision.ModelError`):

- `ErrorKind` — 14 categories: `KindRateLimited`, `KindTimeout`, `KindServerError`, `KindNotImplemented`, `KindServiceUnavailable`, `KindNetwork`, `KindAuthentication`, `KindNotFound`, `KindBadRequest`, `KindContentFilter`, `KindContextTooLarge`, `KindCancelled`, `KindStructuredParse`, `KindUnknown`
- `IsRetryable()` — Quick check for retry logic
- `Unwrap()` — Preserves original cause for `errors.Is` / `errors.AsType`
- Extract via `errors.AsType[*vision.ModelError](err)`

## CLI Usage

```bash
# Build
go build -o vision ./cmd/vision

# Basic analysis
./vision -prompt "Find UI bugs" screenshot.png

# JSON output
./vision -json -prompt "Analyze" screenshot.png

# Streaming
./vision -stream -prompt "Describe this" screenshot.png

# With timeout
./vision -timeout 30s -prompt "Review" screenshot.png
```

## Code Duplication

See [`docs/DUPLICATION_POLICY.md`](docs/DUPLICATION_POLICY.md) for the full
list of extraction helpers and duplication decisions. Current state:
**0 clone groups** at `art-dupl --type-aware -t 1` (verified). Test files and
interface-required signatures are below scan scope by design.
