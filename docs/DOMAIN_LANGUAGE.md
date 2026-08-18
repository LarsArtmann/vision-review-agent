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
| **ModelInfo**          | Catalog-derived model metadata: pricing, context window, capabilities, reasoning flag           | `pkg/vision.ModelInfo`          |
| **Service**            | Catwalk catalog service for model/provider discovery (offline-first, optional remote sync)      | `internal/catalog.Service`      |

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
| **ErrorKind**        | Classified category of a model error (16 kinds). Drives retry vs. fix-input decisions.                          | Enum-like string type                     |
| **CostTracker**      | Thread-safe token accumulator with optional pricing. `CostUSD()` returns 0 without `ModelInfo`.                 | `pkg/vision.CostTracker`                  |

## Error Classification

| Term            | Definition                                                                                               | Context                    |
| --------------- | -------------------------------------------------------------------------------------------------------- | -------------------------- |
| **ModelError**  | A classified error wrapping a provider/context error with an `ErrorKind`. Extracted via `errors.AsType`. | Domain-level error type    |
| **Classify**    | The function that inspects a raw provider error and returns a `*ModelError` with the appropriate `Kind`. | `pkg/errors.Classify`      |
| **IsRetryable** | Quick check: does this error represent a transient failure worth retrying?                               | `ModelError.IsRetryable()` |

## Retry Architecture

The SDK uses a **two-layer retry** design:

### Layer 1: Fantasy HTTP-layer (`Config.MaxRetries`)

- Provider-level retry built into `charm.land/fantasy`
- Retries at the **HTTP transport** level (connection reset, 5xx, 429)
- Default: `0` (disabled)
- Handles low-level transient failures before the error reaches vision code

### Layer 2: Vision-layer (`Config.Retry` / `WithRetry[T]`)

- Domain-level retry with exponential backoff + jitter
- Honors `ModelError.IsRetryable()` — only retries transient errors
- Applied automatically when `Config.Retry *RetryConfig` is set
- `WithRetry[T]` is the explicit middleware wrapper for per-call control
- Default: 3 attempts, capped backoff, jitter on (see `DefaultRetryConfig`)

### Composition

When both layers are active:

1. Fantasy retries first (HTTP-level)
2. If fantasy exhausts retries, the error is classified into a `*ModelError`
3. Vision-layer retry kicks in with backoff + jitter
4. Non-retryable errors (`KindAuthentication`, `KindBadRequest`, etc.) are
   never retried

### Streaming Exclusion

`AnalyzeStream`, `AnalyzeConversationStream`, and `AnalyzeStructuredStream`
do **not** auto-retry. Partial-stream + retry has ambiguous delta semantics
(the caller has already received some chunks). Callers wrap these in
`WithRetry[T]` manually if needed.

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

| Context                       | Description                                                                                                                                                                                      |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Analysis**                  | Core image-analysis lifecycle: validation → preprocessing → model call → hooks → result                                                                                                          |
| **Error Classification**      | Translating provider-specific errors into domain-level `ErrorKind` categories                                                                                                                    |
| **Preprocessing**             | Image transformation (resize, compress) before model invocation                                                                                                                                  |
| **Retry**                     | Two-layer: `Config.MaxRetries` (fantasy HTTP-layer, default 0=disabled) + `Config.Retry` (vision-layer backoff+jitter via `WithRetry[T]`)                                                        |
| **Cost Tracking**             | Token usage accumulation across analysis calls; USD pricing when `ModelInfo` is set                                                                                                              |
| **Model Catalog**             | Catwalk-integrated model/provider discovery (40+ providers, vision model filtering, pricing metadata). Offline-first via embedded data; optional remote sync.                                    |
| **UI Review (visionreviewd)** | Event-sourced screenshot review daemon: watches project goldens, reviews each view with a vision model, projects markdown. See the vocabulary section below.                                     |
| **A2UI**                      | Agent-to-UI protocol support: turns images into declarative, validated A2UI surfaces. See the vocabulary section below.                                                                          |
| **CLI**                       | `parseFlags` parses flags into a `config` struct without calling `os.Exit`, enabling isolated testing. `createProvider` maps provider names to fantasy providers via the catwalk catalog bridge. |

## UI Review Vocabulary (visionreviewd)

| Term           | Definition                                                                                                              | Context                                    |
| -------------- | ----------------------------------------------------------------------------------------------------------------------- | ------------------------------------------ |
| **ViewKey**    | `{Page}--{theme}--{viewport}`, the identity of one reviewed view (parsed from golden file names)                        | `ParseViewKey`                             |
| **Golden**     | A screenshot a project regenerates on change; the daemon's review subject                                               | `testdata/visual/*.png` convention         |
| **Capture**    | The `view.captured` event: a golden's content hash was seen (first time or changed)                                     | Event store, blob store                    |
| **Review**     | The `view.reviewed` event: a vision model judged one capture; carries markdown + score                                  | `Reviewer.Review`                          |
| **Comparison** | The `view.compared` event: model diffed the previous capture (BEFORE) against the new one (AFTER)                       | `Pipeline` auto-compare, `CompareManually` |
| **Pass**       | One scan-review-write cycle over every configured project: scan → skip-seen → blob → compare → review → write           | `Pipeline.Pass`, daemon ticker             |
| **Blob store** | Content-addressed image archive (`<dataDir>/images/<sha256>.<ext>`) keeping BEFORE images alive for compares and replay | `BlobStore`                                |
| **Replay**     | Rebuild the whole markdown projection (views/, comparisons/, INDEX.md) byte-identically from the event journal          | `Replay`                                   |
| **Doctor**     | Preflight check: config paths, glob matches, model endpoint reachability                                                | `visionreviewd doctor`                     |
| **Discover**   | Walk a project for known golden patterns and emit a suggested config                                                    | `visionreviewd discover`                   |
| **INDEX**      | Per-project markdown table of every view with score + trend                                                             | `RenderIndex`                              |

## A2UI Vocabulary

| Term                 | Definition                                                                                                  | Context                            |
| -------------------- | ----------------------------------------------------------------------------------------------------------- | ---------------------------------- |
| **Surface**          | One A2UI UI instance identified by a surfaceId; created, updated, deleted via messages                      | `CreateSurface`, `RootID`          |
| **Catalog**          | The set of component types a client can render (basic catalog: 18 kinds)                                    | `DefaultCatalogID`                 |
| **Adjacency list**   | The flat component model: every component has an id; containers reference children by id                    | `Component`                        |
| **Wire format**      | The JSON messages sent to clients: `{version, <kind>: {...}}` envelopes, one per JSONL line                 | `MarshalJSONL`, `UnmarshalMessage` |
| **Inference format** | The LLM-facing `SurfaceSpec`: one object, props nested under `properties` so the derived schema stays exact | `SurfaceSpec`, `Compile`           |
| **Compile**          | SurfaceSpec → validated wire messages (createSurface, updateComponents, optionally updateDataModel)         | `Compile`                          |
| **Decompile**        | Wire messages → SurfaceSpec; the inverse of Compile (enables edit round-trips)                              | `Decompile`                        |
| **Generate**         | Images → SurfaceSpec → validated messages, via the vision model with structured output                      | `Generate`                         |
| **Dynamic value**    | A property that is a literal, a `{path}` data binding, or a function call                                   | `Bind`, `Literal`                  |

---

> **How to use this file:**
>
> - Keep terms concise — one clear sentence per definition
> - Update when new domain concepts emerge
> - Use these terms consistently in code, docs, and conversations
> - When in doubt about a word's meaning, check here first
