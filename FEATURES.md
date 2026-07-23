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
- **Delegates to Agent** — Analyze, AnalyzeStream, AnalyzeConversation, AnalyzeConversationStream

### CLI

- **Multi-provider** — OpenAI, OpenRouter
- **Streaming** — Real-time text output
- **JSON output** — Machine-readable result format
- **Classified errors** — User-friendly error messages with retry advice

### Infrastructure

- **Nix flake** — Reproducible build environment
- **Table-driven tests** — Pure function tests
- **BDD tests** — User-facing behavior tests (Ginkgo + Gomega)
- **Testify tests** — Error classification and feature tests
- **Mock model** — Comprehensive mock with error injection support

## PARTIALLY DONE

- **Examples** — OpenAI, OpenRouter, and structured examples exist, but no example for conversation, batch, hooks, URL/base64 loading, or structured streaming

## PLANNED

See [ROADMAP.md](ROADMAP.md) for future direction.
