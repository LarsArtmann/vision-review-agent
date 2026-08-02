# Error Design

The vision SDK produces three categories of errors. Understanding which one you
have determines how to inspect, classify, and respond to it.

---

## Taxonomy at a Glance

| Category                   | Type                          | Origin                                                              | Match with                        | Carries                                                     |
| -------------------------- | ----------------------------- | ------------------------------------------------------------------- | --------------------------------- | ----------------------------------------------------------- |
| **Validation sentinel**    | `error` (package-level `var`) | `Config.Validate()`, image loading                                  | `errors.Is(err, sentinel)`        | Fixed message + optional enriched value                     |
| **Classified model error** | `*apperrors.ModelError`       | Model invocation (`Analyze`, `AnalyzeStream`, `AnalyzeStructured*`) | `errors.AsType[*ModelError](err)` | `Kind`, `Op`, `Prompt`, `StatusCode`, `RetryAfter`, `Cause` |
| **Wrapped error**          | `fmt.Errorf("...: %w", err)`  | Internal helpers (`LoadImageFromURL`, `NewAgent`, etc.)             | `errors.Is` traverses `%w` chain  | Context string + underlying cause                           |

---

## 1. Validation Sentinels

**Where:** `pkg/errors/errors.go`, re-exported from `pkg/vision/errors.go`.

Validation sentinels fire **before** any model call — during `Config.Validate()`
(inside `NewAgent`) or during image loading (`LoadImageFromFile`, etc.). They
represent programmer errors: invalid temperature, missing model, empty prompt,
corrupt image data.

### Enrichment Pattern

Ranged validation errors include the offending value via `fmt.Errorf("%w: ...")`:

```go
// pkg/vision/vision.go — Config.Validate()
return fmt.Errorf("%w: got %.2f, want [0.0, 2.0]", ErrInvalidTemperature, c.Temperature)
```

The wrapping preserves `errors.Is` compatibility — the enriched error still
matches the bare sentinel — while adding self-diagnosing context.

### Matching

```go
agent, err := vision.NewAgent(cfg)
if errors.Is(err, vision.ErrInvalidTemperature) {
    // err.Error() includes "got 3.00, want [0.0, 2.0]"
    log.Printf("fix your temperature: %v", err)
}
```

### Full Sentinel List

| Sentinel                     | Trigger                               |
| ---------------------------- | ------------------------------------- |
| `ErrNoModel`                 | `Config.Model` is nil                 |
| `ErrEmptyPrompt`             | Prompt string is empty                |
| `ErrNoImages`                | No images provided                    |
| `ErrInvalidTemperature`      | Temperature outside [0.0, 2.0]        |
| `ErrInvalidMaxTokens`        | MaxOutputTokens is negative           |
| `ErrInvalidTopP`             | TopP outside [0.0, 1.0]               |
| `ErrInvalidTopK`             | TopK is negative                      |
| `ErrInvalidPresencePenalty`  | PresencePenalty outside [-2.0, 2.0]   |
| `ErrInvalidFrequencyPenalty` | FrequencyPenalty outside [-2.0, 2.0]  |
| `ErrInvalidImage`            | Image data doesn't match known format |
| `ErrEmptyImageData`          | Image data is empty                   |
| `ErrImageTooLarge`           | Image exceeds 50 MB                   |

---

## 2. Classified Model Errors (`*ModelError`)

**Where:** `pkg/errors/model.go`, re-exported as `vision.ModelError`.

When the model invocation itself fails (HTTP error, timeout, content filter,
unparseable structured output), the SDK wraps the cause in a `*ModelError`
carrying a domain-level `ErrorKind` and diagnostic metadata.

### Structure

```go
type ModelError struct {
    Kind       ErrorKind      // domain classification (14 categories)
    Op         string         // operation that failed (e.g. "analyze")
    Prompt     string         // user prompt (truncated in Error())
    StatusCode int            // HTTP status (0 if not HTTP)
    RetryAfter time.Duration  // from Retry-After header
    Cause      error          // original provider/context error
}
```

### Matching

```go
result, err := agent.Analyze(ctx, "Find bugs", img)

var me *vision.ModelError
if errors.AsType(err, &me) {
    if me.IsRetryable() {
        // back off and retry
    }
    switch me.Kind {
    case vision.KindRateLimited:
        time.Sleep(me.RetryAfter)
    case vision.KindContentFilter:
        // modify prompt or image
    }
}
```

### ErrorKind Taxonomy

| Kind                     | Retryable | HTTP Status                   | Meaning                              |
| ------------------------ | --------- | ----------------------------- | ------------------------------------ |
| `KindRateLimited`        | yes       | 429                           | Provider rate limit                  |
| `KindTimeout`            | yes       | 408, context.DeadlineExceeded | Request exceeded deadline            |
| `KindServerError`        | yes       | 500, 502, 504                 | Generic 5xx (not 501/503)            |
| `KindServiceUnavailable` | yes       | 503                           | Service temporarily down             |
| `KindOverloaded`         | yes       | 529                           | Anthropic overload signal            |
| `KindNetwork`            | yes       | 0 (transport)                 | Connection reset, EOF, DNS           |
| `KindNotImplemented`     | no        | 501                           | Feature/model doesn't exist          |
| `KindAuthentication`     | no        | 401, 403                      | Bad API key or permissions           |
| `KindPaymentRequired`    | no        | 402                           | Insufficient credits                 |
| `KindNotFound`           | no        | 404                           | Model/resource not found             |
| `KindBadRequest`         | no        | 400                           | Invalid request payload              |
| `KindContentFilter`      | no        | 400 (filtered)                | Content policy rejection             |
| `KindContextTooLarge`    | no        | 400 (too large)               | Input exceeds context window         |
| `KindCancelled`          | no        | —                             | Caller cancelled context             |
| `KindStructuredParse`    | no        | —                             | JSON object generation/parse failure |
| `KindUnknown`            | no        | —                             | Unclassifiable error                 |

### Classification Flow

```
model call error
      │
      ▼
  Classify(err)
      │
      ├── context.Canceled ──────────► KindCancelled
      ├── context.DeadlineExceeded ──► KindTimeout
      ├── *fantasy.NoObjectGeneratedError ► KindStructuredParse
      ├── *fantasy.ProviderError ────► classifyProviderError()
      │       │
      │       ├── IsContextTooLarge ─► KindContextTooLarge
      │       ├── 401/403 ───────────► KindAuthentication
      │       ├── 402 ───────────────► KindPaymentRequired
      │       ├── 404 ───────────────► KindNotFound
      │       ├── 429 ───────────────► KindRateLimited
      │       ├── 408 ───────────────► KindTimeout
      │       ├── 501 ───────────────► KindNotImplemented
      │       ├── 503 ───────────────► KindServiceUnavailable
      │       ├── 529 ───────────────► KindOverloaded
      │       ├── 400 + content filter signals ► KindContentFilter
      │       ├── 400 (other) ───────► KindBadRequest
      │       ├── 5xx (other) ───────► KindServerError
      │       ├── status=0 + retryable ► KindNetwork
      │       └── fallback ──────────► KindUnknown
      │
      └── fallback ──────────────────► KindUnknown
```

The `KindStructuredParse` kind is also set explicitly via `apperrors.Wrap()`
when the model returns a response whose JSON does not unmarshal into the
target type `T` (both streaming and non-streaming paths).

---

## 3. Wrapped Errors

Internal helper functions wrap errors with operational context using
`fmt.Errorf("operation: %w", err)`. These are not sentinels or `*ModelError`,
but the `%w` verb preserves the error chain so `errors.Is` and `errors.AsType`
traverse through them.

```go
// LoadImageFromURL wraps download failures with the URL:
img, err := vision.LoadImageFromURL(ctx, url)
// err = `download image from "https://...": <underlying>`

// errors.Is traverses the chain:
if errors.Is(err, vision.ErrInvalidImage) {
    // the URL is in the message, but the sentinel still matches
}
```

---

## Consumer Decision Matrix

| Question                                  | Answer                                             |
| ----------------------------------------- | -------------------------------------------------- |
| Was it a config or input error?           | `errors.Is(err, sentinel)`                         |
| Was it a model invocation error?          | `errors.AsType[*ModelError](err)`                  |
| Should I retry?                           | `me.IsRetryable()` or `apperrors.IsRetryable(err)` |
| How long should I wait?                   | `me.RetryAfter`                                    |
| What HTTP status did the provider return? | `me.StatusCode`                                    |
| What was the operation?                   | `me.Op`                                            |
| What prompt triggered it?                 | `me.Prompt`                                        |

---

## See Also

- `pkg/errors/errors.go` — sentinel definitions
- `pkg/errors/model.go` — `ModelError`, `ErrorKind`, `Classify`, `Wrap`, `IsRetryable`
- `pkg/vision/errors.go` — re-exports for consumers
- `examples/error-handling/main.go` — end-to-end error handling example
