# Feature Inventory

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
- **LoadImageFromURL** — Download from URL via HTTP (respects context, Content-Type detection)
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
- **MaxRetries** — Transient error retry count
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

### Hooks & Observability

- **Hooks across all methods** — OnStart/OnFinish/OnError fire in Analyze, AnalyzeStream, AnalyzeConversation, AnalyzeConversationStream, AnalyzeStructured, AnalyzeStructuredStream
- **CostTracker** — Thread-safe accumulator for token usage across calls; integrates with Hooks.OnFinish

### Reliability & Preprocessing

- **WithRetry[T]** — Generic retry middleware with exponential backoff + jitter, respects IsRetryable()
- **RetryConfig** — Configurable attempts, initial/max backoff, multiplier, jitter
- **ResizeImage** — High-quality Catmull-Rom resize (longest-side cap), PNG/JPEG/WebP/GIF decode
- **LoadImageFromURLWithClient** — Custom `*http.Client` for proxies/timeouts/TLS
- **URL image validation** — LoadImageFromURL rejects non-image bodies via ValidateImage

### Advanced Capabilities (fantasy passthrough)

- **Tools / ToolChoice** — Typed tool/function calling wired to fantasy.WithTools / WithToolChoice
- **StopConditions** — Composable agent-loop termination (StepCountIs, HasToolCall, MaxTokensUsed)
- **PrepareStep** — Per-step interceptor for model/prompt/tool mutation
- **Headers / UserAgent** — Extra HTTP headers and User-Agent on provider requests

### CLI

- **Multi-provider** — OpenAI, OpenRouter, Anthropic, Google (ADC), openaicompat (local models)
- **Streaming** — Real-time text output
- **JSON output** — Machine-readable result format
- **-structured** — Built-in UIReview schema with structured JSON output
- **Classified errors** — User-friendly error messages with retry advice

### Infrastructure

- **Nix flake** — Reproducible build environment
- **Table-driven tests** — Pure function tests
- **BDD tests** — User-facing behavior tests (Ginkgo + Gomega)
- **Testify tests** — Error classification and feature tests
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

## PLANNED

See [ROADMAP.md](ROADMAP.md) for future direction.
