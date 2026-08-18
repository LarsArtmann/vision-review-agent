# Feature Inventory

Honest inventory of what exists and how well it works. For future direction,
see [ROADMAP.md](ROADMAP.md); for open work, see [TODO_LIST.md](TODO_LIST.md).

> **Status vocabulary:** **DONE** = code present and working (tests pass).
> **PARTIALLY DONE** = ships but has known gaps, edge-case bugs, or missing
> wiring. **PLANNED** = designed/documented but no code exists.

---

## DONE

### Core Analysis

- **Analyze** — Single-request image analysis with prompt
- **AnalyzeStream** — Real-time text streaming via callback
- **AnalyzeStructured[T]** — Typed structured output via JSON schema
- **AnalyzeStructuredStream[T]** — Stream partial typed objects as they arrive
- **AnalyzeConversation** — Multi-turn analysis with conversation history
- **AnalyzeConversationStream** — Streaming multi-turn analysis
- **AnalyzeBatch** — Concurrent analysis of multiple images with bounded parallelism (per-image error capture + classification + timing via `BatchResult.Duration`)

### Image Loading

- **LoadImageFromFile** — Load from filesystem path (auto-detects media type)
- **LoadImageFromReader** — Load from any `io.Reader` (50 MB limit)
- **LoadImageFromURL** — Download from URL via HTTP (respects context, Content-Type detection, validates image format post-download)
- **LoadImageFromURLWithClient** — Custom `*http.Client` variant (proxies, timeouts, TLS)
- **LoadImageFromBase64** — Decode base64 string (supports standard, URL-safe, raw encodings)
- **NewImageSource** — Construct directly from bytes with validation
- **DetectImageFormat** — Magic byte detection (PNG, JPEG, GIF, WebP, BMP)
- **ValidateImage** — Validate image data against known magic bytes

### Image Preprocessing

- **PreprocessConfig** — `Config.Preprocess` auto-resizes (and re-encodes JPEG) before every `Analyze*` call
- **PreprocessImage** — Apply a `PreprocessConfig` to an `ImageSource` (called automatically by the Agent)
- **ResizeImage** — Catmull-Rom downscale, aspect-preserving, longest-side cap; decodes PNG/JPEG/GIF/WebP/BMP
- **ResizeImageWithQuality** — `ResizeImage` with caller-controlled JPEG quality (1-100)
- **CompressImage** — Re-encode JPEG at lower quality without resizing; preserves PNG format
- **ScreenshotAnalyzer.WithMaxDimension** — Fluent builder that sets `Config.Preprocess`

### Configuration

- **Config.Validate** — Full validation of all parameters at construction
- **SystemPrompt** — Optional system prompt
- **Temperature** — Sampling randomness (0.0-2.0)
- **TopP** — Nucleus sampling (0.0-1.0)
- **TopK** — Top-k sampling limit
- **PresencePenalty** — Token presence penalty (-2.0 to 2.0)
- **FrequencyPenalty** — Token frequency penalty (-2.0 to 2.0)
- **MaxOutputTokens** — Response length limit
- **MaxRetries** — Fantasy HTTP-layer retry count (0 disables)
- **Retry** — Vision-layer `*RetryConfig` (backoff + jitter) auto-applied to all non-streaming analysis methods; composes with `MaxRetries`
- **RequestTimeout** — Per-request timeout
- **Preprocess** — `*PreprocessConfig` auto-applied before every `Analyze*` call
- **Hooks** — Lifecycle callbacks (OnStart, OnFinish, OnError), fire in every analysis path

### Retry & Cost

- **WithRetry[T]** — Generic external retry middleware (exponential backoff + jitter, honors `IsRetryable()`)
- **DefaultRetryConfig** — Sensible defaults (3 attempts, capped backoff, jitter on)
- **CostTracker** — Thread-safe token accumulator (`Add`, `AddResult`, `Total`, `Calls`, `Reset`)
- **NewAgentWithCostTracker** — Convenience constructor that auto-wires `CostTracker` into `Hooks.OnFinish`, composing with user-supplied hooks

### Error Handling

- **ModelError** — Classified errors with ErrorKind, StatusCode, RetryAfter, IsRetryable(), Unwrap()
- **ErrorKind** — 16 error categories (rate-limited, timeout, server-error, service-unavailable, overloaded, network, authentication, payment-required, not-found, bad-request, content-filter, not-implemented, context-too-large, cancelled, structured-parse, unknown)
- **Classify** — Automatic error classification from provider responses (HTTP status + content-filter signal detection from verified provider messages)
- **IsRetryable** — Quick retryability check
- **Sentinel errors** — 12 validation sentinel errors (ErrNoModel, ErrEmptyPrompt, etc.)

### ScreenshotAnalyzer

- **Fluent builder** — Chainable configuration (WithSystemPrompt, WithTemperature, etc.)
- **All model parameters** — TopP, TopK, PresencePenalty, FrequencyPenalty
- **Cache invalidation** — All builder methods invalidate cached agent
- **WithHooks** — Set lifecycle callbacks via the fluent builder
- **WithMaxDimension** — Set `Config.Preprocess` via the builder
- **Delegates to Agent** — Analyze, AnalyzeStream, AnalyzeConversation, AnalyzeConversationStream

### Advanced Capabilities (fantasy passthrough)

- **Tools / ToolChoice** — Typed tool/function calling wired to fantasy.WithTools / WithToolChoice
- **StopConditions** — Composable agent-loop termination (StepCountIs, HasToolCall, MaxTokensUsed)
- **PrepareStep** — Per-step interceptor for model/prompt/tool mutation
- **Headers / UserAgent** — Extra HTTP headers and User-Agent on provider requests

### Infrastructure

- **Nix flake** — Reproducible build environment with `nix run .#test`, `nix run .#lint`, `nix flake check`
- **CI workflow** — `.github/workflows/ci.yml` (build, vet, race test, coverage gate, lint, format check, nix flake check, actionlint)
- **Table-driven tests** — Pure function tests
- **BDD tests** — User-facing behavior tests (Ginkgo + Gomega)
- **Testify tests** — Error classification and feature tests
- **Fuzz tests** — `FuzzDetectImageFormat`, `FuzzDecodeBase64Flex`, `FuzzEncodeImage`, `FuzzResizeImage`, `FuzzCompressImage`, `FuzzParseFlags`
- **Mock model** — Comprehensive mock with error injection + call counting

### Model Catalog (catwalk integration)

- **40+ providers** — Built-in catalog via `charm.land/catwalk` embedded data (OpenAI, Anthropic, Google Gemini, OpenRouter, xAI, and 35+ more)
- **Model discovery** — `Service.FindModel`, `FindModelInProvider`, `VisionModels` for querying the catalog
- **Model suggestions** — Levenshtein-distance typo correction ("did you mean gpt-4o?")
- **Provider info** — `-list-providers`, `-list-models`, `-provider-info` listing flags
- **Remote sync** — Optional ETag-based catalog updates via `CATWALK_URL` env var (5s timeout, falls back to cache then embedded)
- **Provider bridge** — `BuildProvider(catwalk.Provider) → fantasy.Provider` maps catalog metadata to fantasy constructors

### Cost Tracking (pricing-aware)

- **ModelInfo** — Catalog-derived model metadata (context window, pricing, capabilities, reasoning flag)
- **CostTracker.SetPricing** — Per-1M-token pricing wired from `Config.ModelInfo`
- **CostTracker.CostUSD** — USD cost calculation from accumulated token usage
- **NewAgentWithCostTracker** — Auto-wires pricing from `Config.ModelInfo` when available
- **Backward compatible** — Without `ModelInfo`, `CostUSD()` returns 0 (no behavior change)

### CLI

- **Multi-provider** — All 40+ catwalk providers plus `openaicompat` for local servers (Ollama, LM Studio)
- **Provider alias** — `-provider google` normalizes to catwalk's `gemini` ID
- **Streaming** — Real-time text output
- **JSON output** — Machine-readable result format
- **-structured** — Built-in UIReview schema with structured JSON output
- **Classified errors** — User-friendly error messages with per-kind retry advice
- **Model suggestions** — Typo detection with "did you mean?" hints for unknown models
- **Catalog listing** — `-list-providers`, `-list-models [-provider X]`, `-provider-info` flags
- **CLI tests** — `adviceForKind`, `buildConfig`, `parseTimeout`, `createProvider` error paths, alias normalization, provider bridge integration

### visionreviewd (UI review daemon)

Event-sourced daemon in `internal/reviewd` + `cmd/visionreviewd` that watches
configured screenshot globs, reviews changes with an OpenAI-compatible local
model (llama-server), writes Crush-consumable markdown, and records everything
as go-cqrs-lite events on bbolt.

- **Scan & pipeline** — `ScanProject` globs → dedupe by SHA-256 → blob-archive →
  auto BEFORE→AFTER compare → review → markdown + INDEX (`internal/reviewd/pipeline.go`)
- **Event sourcing** — `view.captured` / `view.reviewed` / `view.compared` on
  per-view streams; folded `ViewState` with score trend (bbolt store)
- **Content-addressed blob store** — `<dataDir>/images/<sha256>.<ext>`, so
  overwritten goldens stay comparable after the fact
- **Markdown output** — per-view reviews, timestamped comparisons, project
  INDEX with score table and trend arrows; all writes atomic
- **Daemon loop** — immediate pass + ticker, logged-and-continue per-pass
  failures, clean SIGINT/SIGTERM shutdown (`internal/reviewd/daemon.go`)
- **7 subcommands** — run, once, discover, compare, events, replay, doctor,
  version (`cmd/visionreviewd`)
- **Replay** — rebuild the whole reviews directory byte-identically from the
  event journal (`internal/reviewd/replay.go`); INDEX stamps derive from row
  timestamps, not wall clock, so pass and replay agree
- **events command** — journal listing with `-project/-view/-type/-last`
  filters carrying hashes and scores
- **doctor** — config, dir writability, glob match counts, and
  `{baseUrl}/models` model-listing check; exit code reflects failures
- **E2E confidence** — Review and Compare verified through the real
  openaicompat provider against a fake OpenAI-compatible httptest server,
  including image-part counts and score parsing (`internal/reviewd/fakeserver_test.go`)
- **Nix packaging** — `packages.visionreviewd` buildGoModule with version
  ldflags and `GOEXPERIMENT=jsonv2` (go-cqrs-lite imports `encoding/json/v2`)
- **NixOS module** — `nixosModules.visionreviewd`: hardened DynamicUser
  service plus optional, default-disabled llama-server unit (port 8390);
  SystemNix ships a lazy wrapper (see `docs/visionreviewd-systemnix.md`)

Real-model bring-up and host activation are tracked in
[TODO_LIST.md](TODO_LIST.md).

### A2UI Support (pkg/vision/a2ui)

- **Vision-driven A2UI generation** — `a2ui.Generate` turns screenshots and
  mockups into complete, validated A2UI surfaces via any vision model
  (`AnalyzeStructured` + catalog-grounded prompt)
- **Inference format + compiler** — `SurfaceSpec` (LLM-facing, schema-derived)
  compiled by `a2ui.Compile` into the canonical wire sequence
  (`createSurface` → `updateComponents` → `updateDataModel`)
- **Wire types + JSONL codec** — the four v0.9.1 message kinds as a
  tagged-union `Message` interface, plus `MarshalJSONL`/`UnmarshalJSONL` for
  the JSON Lines transport encoding
- **Structural validation** — `a2ui.Validate`/`Issues`: root presence, unique
  IDs, resolvable child references, acyclicity, surface lifecycle ordering,
  envelope versions; typed sentinels (`ErrValidation`, `ErrComponentCycle`,
  `ErrMalformedMessage`)
- **Official-schema conformance** — compiled output, hand-built messages of
  all four kinds, and every builder are validated against the pinned official
  v0.9.1 schemas in `conformance_test.go` (positive control included)
- **Typed components + builders** — `Component` adjacency-list nodes with
  static/dynamic `ChildList`, accessibility labels, `Bind`/`Literal` dynamic
  values, and a constructor for every kind in the basic catalog (18/18; every
  builder's wire output is schema-checked in
  `TestAllBuildersConformToOfficialSchema`):

| Kind                   | Constructor                                           |
| ---------------------- | ----------------------------------------------------- |
| Text                   | `NewText`                                             |
| Button                 | `NewButton`                                           |
| Column / Row / List    | `NewColumn` / `NewRow` / `NewList`                    |
| Card / Modal           | `NewCard` / `NewModal`                                |
| Image / Icon / Divider | `NewImage` / `NewIcon` / `NewDivider`                 |
| Tabs                   | `NewTabs` (takes `Tab{Title, ChildID}` values)        |
| TextField / CheckBox   | `NewTextField` / `NewCheckBox`                        |
| ChoicePicker           | `NewChoicePicker` (takes `ChoicePickerOption` values) |
| DateTimeInput / Slider | `NewDateTimeInput` / `NewSlider`                      |
| AudioPlayer / Video    | `NewAudioPlayer` / `NewVideo`                         |

- **Decompile** — `a2ui.Decompile` folds a wire message stream back into a
  `SurfaceSpec` (RFC 6901 pointer writes/removes for data-model edits),
  closing the `Compile` asymmetry for diffing and edit round-trips
- **Theme + DataModel options** — `GenerateOptions.Theme` passes through when
  the model emitted none; `GenerateOptions.DataModel` seeds the data model
  with model keys winning
- **Spec version** — v0.9.1 emitted, v0.9 accepted on input; components from
  the A2UI basic catalog

### Examples

- **openai** — Basic OpenAI vision analysis
- **openrouter** — OpenRouter multi-model routing
- **structured** — Typed structured output (UIReview)
- **conversation** — Multi-turn conversation with history
- **batch** — Concurrent batch analysis with bounded parallelism
- **hooks** — Lifecycle callbacks for logging/metrics
- **structured-stream** — Structured streaming with partial objects
- **url-loading** — Load from URL (custom client) and base64 round-trip
- **error-handling** — Classified error handling with kind-to-action lookup
- **a2ui** — Screenshot → A2UI surface as JSON Lines

## PARTIALLY DONE

> These ship and compile, but have known gaps. See [TODO_LIST.md](TODO_LIST.md)
> for the specific fixes tracked.

- **Hooks across structured methods** — `OnStart`/`OnFinish`/`OnError` fire in
  `AnalyzeStructured` / `AnalyzeStructuredStream`, but `OnFinish` receives a
  **synthesized** `*AnalyzeResult` with nil `RawResponse` (documented contract,
  but a nil-pointer hazard for consumers that dereference it) and `Text`
  holding raw JSON, not prose. Hooks in `Analyze`/`AnalyzeStream`/
  `AnalyzeConversation` are fully DONE. A proper `HooksEvent` redesign is a
  breaking change deferred to ROADMAP.
- **Streaming auto-retry** — Deliberately excluded. `AnalyzeStream`,
  `AnalyzeConversationStream`, and `AnalyzeStructuredStream` do NOT auto-retry
  on `Config.Retry` (partial-stream + retry has ambiguous delta semantics).
  Documented; callers wrap in `WithRetry` manually if needed.

## PLANNED

See [ROADMAP.md](ROADMAP.md) for future direction and open questions.
