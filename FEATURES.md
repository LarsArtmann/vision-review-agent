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
- **AnalyzeBatch** — Concurrent analysis of multiple images with bounded parallelism (per-image error capture + classification)

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

- **ModelError** — Classified errors with ErrorKind, StatusCode, IsRetryable(), Unwrap()
- **ErrorKind** — 14 error categories (rate-limited, timeout, server-error, service-unavailable, network, authentication, not-found, bad-request, content-filter, not-implemented, context-too-large, cancelled, structured-parse, unknown)
- **Classify** — Automatic error classification from provider responses (HTTP status + content-filter signal detection)
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
- **CI workflow** — `.github/workflows/ci.yml` (build, vet, race test, coverage gate, lint, format check)
- **Table-driven tests** — Pure function tests
- **BDD tests** — User-facing behavior tests (Ginkgo + Gomega)
- **Testify tests** — Error classification and feature tests
- **Fuzz tests** — `FuzzDetectImageFormat`, `FuzzDecodeBase64Flex`
- **Mock model** — Comprehensive mock with error injection + call counting

### CLI

- **Multi-provider** — OpenAI, OpenRouter (DONE); Anthropic, Google (ADC), openaicompat (PARTIALLY DONE — build-verified only)
- **Streaming** — Real-time text output
- **JSON output** — Machine-readable result format
- **-structured** — Built-in UIReview schema with structured JSON output
- **Classified errors** — User-friendly error messages with per-kind retry advice
- **CLI tests** — `adviceForKind`, `buildConfig`, `parseTimeout`, `createProvider` error paths

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
- **CLI providers (Anthropic, Google, openaicompat)** — Compile and appear in
  `-h`, but are **build-verified only** — no runtime credentials tested. Google
  uses ADC (environment-specific); openaicompat expects a local server.
- **Streaming auto-retry** — Deliberately excluded. `AnalyzeStream`,
  `AnalyzeConversationStream`, and `AnalyzeStructuredStream` do NOT auto-retry
  on `Config.Retry` (partial-stream + retry has ambiguous delta semantics).
  Documented; callers wrap in `WithRetry` manually if needed.

## PLANNED

See [ROADMAP.md](ROADMAP.md) for future direction and open questions.
