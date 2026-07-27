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
- **AnalyzeBatch** — Concurrent analysis of multiple images with bounded parallelism

### Image Loading

- **LoadImageFromFile** — Load from filesystem path (auto-detects media type)
- **LoadImageFromReader** — Load from any `io.Reader` (50 MB limit)
- **LoadImageFromURL** — Download from URL via HTTP (respects context, Content-Type detection, validates image format post-download)
- **LoadImageFromURLWithClient** — Custom `*http.Client` variant (proxies, timeouts, TLS)
- **LoadImageFromBase64** — Decode base64 string (supports standard, URL-safe, raw encodings)
- **NewImageSource** — Construct directly from bytes with validation
- **DetectImageFormat** — Magic byte detection (PNG, JPEG, GIF, WebP, BMP)
- **ValidateImage** — Validate image data against known magic bytes

### Configuration

- **Config.Validate** — Full validation of all parameters at construction
- **SystemPrompt** — Optional system prompt
- **Temperature** — Sampling randomness (0.0-2.0)
- **TopP** — Nucleus sampling (0.0-1.0)
- **TopK** — Top-k sampling limit
- **PresencePenalty** — Token presence penalty (-2.0 to 2.0)
- **FrequencyPenalty** — Token frequency penalty (-2.0 to 2.0)
- **MaxOutputTokens** — Response length limit
- **MaxRetries** — Transient error retry count (fantasy HTTP-level retry)
- **RequestTimeout** — Per-request timeout
- **Hooks** — Lifecycle callbacks (OnStart, OnFinish, OnError)

### Error Handling

- **ModelError** — Classified errors with ErrorKind, StatusCode, IsRetryable()
- **ErrorKind** — 11 error categories (rate-limited, timeout, auth, etc.)
- **Classify** — Automatic error classification from provider responses
- **IsRetryable** — Quick retryability check
- **Sentinel errors** — 12 validation sentinel errors (ErrNoModel, ErrEmptyPrompt, etc.)

### ScreenshotAnalyzer

- **Fluent builder** — Chainable configuration (WithSystemPrompt, WithTemperature, etc.)
- **All model parameters** — TopP, TopK, PresencePenalty, FrequencyPenalty
- **Cache invalidation** — Builder methods invalidate cached agent (bug fix)
- **WithHooks** — Set lifecycle callbacks via the fluent builder
- **Delegates to Agent** — Analyze, AnalyzeStream, AnalyzeConversation, AnalyzeConversationStream

### Infrastructure

- **Nix flake** — Reproducible build environment with `nix run .#test` and `nix run .#lint`
- **Table-driven tests** — Pure function tests
- **BDD tests** — User-facing behavior tests (Ginkgo + Gomega)
- **Testify tests** — Error classification and feature tests
- **Fuzz tests** — `FuzzDetectImageFormat`, `FuzzDecodeBase64Flex`
- **Mock model** — Comprehensive mock with error injection support

### Examples

- **openai** — Basic OpenAI vision analysis
- **openrouter** — OpenRouter multi-model routing
- **structured** — Typed structured output (UIReview)
- **conversation** — Multi-turn conversation with history
- **batch** — Concurrent batch analysis with bounded parallelism
- **hooks** — Lifecycle callbacks for logging/metrics
- **structured-stream** — Structured streaming with partial objects
- **url-loading** — Load from URL (custom client) and base64 round-trip

## PARTIALLY DONE

> These ship and compile, but have known gaps. See [TODO_LIST.md](TODO_LIST.md)
> for the specific fixes tracked.

- **Hooks across structured methods** — `OnStart`/`OnFinish`/`OnError` fire in
  `AnalyzeStructured` / `AnalyzeStructuredStream`, but `OnFinish` receives a
  **synthesized** `*AnalyzeResult` with nil `RawResponse` (nil-pointer hazard
  for consumers) and `Text` holding raw JSON, not prose. Hooks in
  `Analyze`/`AnalyzeStream`/`AnalyzeConversation` are fully DONE.
- **WithRetry[T] / RetryConfig** — Generic retry middleware with exponential
  backoff + jitter works, but **conflicts conceptually with `Config.MaxRetries`**
  (two retry systems, undocumented relationship). Not wired into
  `AnalyzeBatch` / `AnalyzeConversation`.
- **CostTracker** — Thread-safe token accumulator works, but is **detached** —
  no `Agent` integration; caller must manually wire `Hooks.OnFinish`.
- **ResizeImage** — High-quality Catmull-Rom resize works for PNG/JPEG/WebP/GIF,
  but is an **island feature** (never called by Agent; no `Config.Preprocess`
  wiring), has **no compress/convert**, and **cannot decode BMP** despite
  `MediaTypeBMP` existing.
- **MediaTypeBMP** — Constant exists and `DetectImageFormat` recognizes BMP
  magic bytes, but `mediaTypeFromExtension` does **not** map `.bmp` →
  `MediaTypeBMP` (falls back to `MediaTypePNG`), and no BMP decoder is
  registered for `ResizeImage`.
- **CLI providers (Anthropic, Google, openaicompat)** — Compile and appear in
  `-h`, but are **build-verified only** — no runtime credentials tested. Google
  uses ADC (environment-specific); openaicompat expects a local server.

### Advanced Capabilities (fantasy passthrough) — DONE

- **Tools / ToolChoice** — Typed tool/function calling wired to fantasy.WithTools / WithToolChoice
- **StopConditions** — Composable agent-loop termination (StepCountIs, HasToolCall, MaxTokensUsed)
- **PrepareStep** — Per-step interceptor for model/prompt/tool mutation
- **Headers / UserAgent** — Extra HTTP headers and User-Agent on provider requests

### CLI — DONE (except providers noted above)

- **Multi-provider** — OpenAI, OpenRouter (DONE); Anthropic, Google (ADC), openaicompat (PARTIALLY DONE)
- **Streaming** — Real-time text output
- **JSON output** — Machine-readable result format
- **-structured** — Built-in UIReview schema with structured JSON output
- **Classified errors** — User-friendly error messages with retry advice

## PLANNED

See [ROADMAP.md](ROADMAP.md) for future direction and open questions.
