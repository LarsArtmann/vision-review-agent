# Domain Language

A **Unified Language** for Vision Review Agent — shared across Product Owner,
Developer, and AI. Inspired by Domain-Driven Design (DDD) Ubiquitous Language.

Every term below should mean the **same thing** to everyone who reads it.
If a word means something different to a developer than to a customer, define it here.

## Glossary

| Term                   | Definition                                                                                      | Context                         |
| ---------------------- | ----------------------------------------------------------------------------------------------- | ------------------------------- |
| **Agent**              | The core type that analyzes images using a vision-capable language model                        | `pkg/vision.Agent`              |
| **ImageSource**        | A validated image (data + media type + filename) ready for analysis                             | `pkg/vision.ImageSource`        |
| **Analysis**           | A single model invocation that sends images + a prompt and receives a text or structured result | User-facing concept             |
| **AnalyzeResult**      | The outcome of a text analysis: response text, token usage, and raw provider result             | `pkg/vision.AnalyzeResult`      |
| **ScreenshotAnalyzer** | A fluent builder for screenshot-analysis use cases; wraps `Agent` with UI-focused defaults      | `pkg/vision.ScreenshotAnalyzer` |
| **Conversation**       | Multi-turn history accumulator for follow-up questions with prior context                       | `pkg/vision.Conversation`       |

## Entities

Objects with identity and lifecycle:

| Term                   | Definition                                                                             | Context                           |
| ---------------------- | -------------------------------------------------------------------------------------- | --------------------------------- |
| **Agent**              | Owns a `Config` and an underlying `fantasy.Agent`. Created once, reused across calls.  | Single per analysis configuration |
| **ScreenshotAnalyzer** | Holds a cached `Agent` that is rebuilt when any `With*` builder method changes config. | Builder pattern, cache-safe       |

## Value Objects

Immutable objects defined by their attributes:

| Term                 | Definition                                                                                                      | Context                                   |
| -------------------- | --------------------------------------------------------------------------------------------------------------- | ----------------------------------------- |
| **ImageSource**      | `Data []byte` + `MediaType` + `Filename`. Created via constructors that validate non-empty data.                | Immutable after construction              |
| **Config**           | All agent configuration: model, prompts, sampling params, hooks, retry, preprocessing. Validated at `NewAgent`. | Value object, copied by the agent         |
| **MediaType**        | Defined string type for image formats: PNG, JPEG, GIF, WebP, BMP.                                               | Type safety over raw strings              |
| **PreprocessConfig** | Image preprocessing settings: max dimension, JPEG quality. Zero-value disables.                                 | Applied automatically by the agent        |
| **RetryConfig**      | Retry policy: max attempts, initial backoff, cap, multiplier, jitter. Zero-value falls back to defaults.        | Used by `Config.Retry` and `WithRetry[T]` |
| **Usage**            | Token usage from a single call: input, output, total.                                                           | Accumulated by `CostTracker`              |
| **ErrorKind**        | Classified category of a model error (14 kinds). Drives retry vs. fix-input decisions.                          | Enum-like string type                     |

## Error Classification

| Term            | Definition                                                                                               | Context                    |
| --------------- | -------------------------------------------------------------------------------------------------------- | -------------------------- |
| **ModelError**  | A classified error wrapping a provider/context error with an `ErrorKind`. Extracted via `errors.AsType`. | Domain-level error type    |
| **Classify**    | The function that inspects a raw provider error and returns a `*ModelError` with the appropriate `Kind`. | `pkg/errors.Classify`      |
| **IsRetryable** | Quick check: does this error represent a transient failure worth retrying?                               | `ModelError.IsRetryable()` |

## Events

Things that happen in the domain:

| Term         | Definition                                                                                                       | Context          |
| ------------ | ---------------------------------------------------------------------------------------------------------------- | ---------------- |
| **OnStart**  | Hook fired before analysis begins, after validation passes. Receives prompt + image count.                       | `Hooks.OnStart`  |
| **OnFinish** | Hook fired after a successful analysis. Receives `*AnalyzeResult` (RawResponse may be nil for structured calls). | `Hooks.OnFinish` |
| **OnError**  | Hook fired when analysis fails with a non-validation error. Receives the classified `*ModelError`.               | `Hooks.OnError`  |

## Commands

Actions the system performs:

| Term                       | Definition                                                                              | Context                         |
| -------------------------- | --------------------------------------------------------------------------------------- | ------------------------------- |
| **Analyze**                | Send images + prompt → get text analysis.                                               | `Agent.Analyze`                 |
| **AnalyzeStream**          | Same as Analyze but streams text chunks via a callback.                                 | `Agent.AnalyzeStream`           |
| **AnalyzeStructured[T]**   | Send images + prompt → get typed JSON result. Generic, package-level.                   | `vision.AnalyzeStructured[T]`   |
| **AnalyzeConversation**    | Analyze with multi-turn conversation history.                                           | `Agent.AnalyzeConversation`     |
| **AnalyzeBatch**           | Analyze many images concurrently with bounded concurrency.                              | `Agent.AnalyzeBatch`            |
| **ResizeImage**            | Downscale an image to a max dimension (Catmull-Rom, aspect-preserving).                 | `vision.ResizeImage`            |
| **ResizeImageWithQuality** | Like ResizeImage but lets the caller control JPEG encoding quality (1-100).             | `vision.ResizeImageWithQuality` |
| **CompressImage**          | Re-encode an image to reduce byte size without resizing. Returns original if no shrink. | `vision.CompressImage`          |
| **WithRetry[T]**           | Wrap any analysis call with exponential-backoff retry on transient errors.              | `vision.WithRetry[T]`           |

## Bounded Contexts

| Context                  | Description                                                                                                                                                       |
| ------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Analysis**             | Core image-analysis lifecycle: validation → preprocessing → model call → hooks → result                                                                           |
| **Error Classification** | Translating provider-specific errors into domain-level `ErrorKind` categories                                                                                     |
| **Preprocessing**        | Image transformation (resize, compress) before model invocation                                                                                                   |
| **Retry**                | Two-layer: `Config.MaxRetries` (fantasy HTTP-layer, default 0=disabled) + `Config.Retry` (vision-layer backoff+jitter via `WithRetry[T]`)                         |
| **Cost Tracking**        | Token usage accumulation across analysis calls                                                                                                                    |
| **CLI**                  | `parseFlags` parses flags into a `config` struct without calling `os.Exit`, enabling isolated testing. `createProvider` maps provider names to fantasy providers. |

---

> **How to use this file:**
>
> - Keep terms concise — one clear sentence per definition
> - Update when new domain concepts emerge
> - Use these terms consistently in code, docs, and conversations
> - When in doubt about a word's meaning, check here first
