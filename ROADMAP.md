# Roadmap

Long-term direction and raw ideas not yet refined into actionable tasks.
For short-term actionable work, see [TODO_LIST.md](TODO_LIST.md).
For current feature inventory, see [FEATURES.md](FEATURES.md).

> Items graduate from here to `TODO_LIST.md` when they become bounded and
> actionable. Items that shipped move to [CHANGELOG.md](CHANGELOG.md) and leave
> this file.

---

## Near-term direction

- **Auto-wired preprocessing pipeline** — `ResizeImage` shipped standalone; the
  next step is a composable `Config.Preprocess` (max dimension, quality, format
  conversion) applied automatically inside every `Analyze*`, plus
  `ScreenshotAnalyzer.WithMaxDimension`. Goal: reduce token usage and support
  provider-specific limits without callers wrapping every call.
- **Unified retry strategy** — reconcile `Config.MaxRetries` (fantasy HTTP retry)
  with `WithRetry[T]` / `RetryConfig` (external wrapper). One system, documented,
  wired into `AnalyzeBatch` / `AnalyzeConversation`. See Open questions.
- **Structured hooks done right** — replace the synthesized `*AnalyzeResult`
  hack with a proper discriminated payload (`HooksEvent` with a `Kind`) or
  `StructuredHooks[T]`. See Open questions.
- **catwalk integration for CLI** — replace hand-rolled CLI providers
  (Anthropic, Google, openaicompat) with `charmbracelet/catwalk` to avoid rot
  as fantasy adds providers. See Open questions.

## Mid-term ideas

- **Provider-defined tools** — expose `WithProviderDefinedTools` for
  provider-native tools like web search and computer use (typed tools are
  already exposed via `Config.Tools`).
- **Context-aware batch** — let batch analysis share context or conversation
  state across images.
- **Batch-level hooks** — `Hooks.OnBatchStart` / `OnBatchFinish` for
  batch-scoped observability (per-image hooks already fire via internal
  `Analyze`).
- **Built-in cost tracking** — `Agent.Cost()` method or automatic
  `CostTracker` wiring via `Hooks.OnFinish` (the standalone `CostTracker` type
  already exists).
- **Typed config-validation errors** — move beyond sentinel errors to typed
  validation failures that carry the field name, the invalid value, and the
  allowed range.
- **Custom HTTP client for providers** — `LoadImageFromURLWithClient` exists
  for image loading; extend the pattern to provider connections (proxies,
  timeouts, TLS).
- **Provider failover** — automatically try a secondary provider if the primary
  fails with a retryable error.
- **EXIF stripping** — strip EXIF metadata during preprocessing for privacy.

## Long-term ideas

- **Result caching** — cache analysis results by image hash + prompt to avoid
  redundant API calls.
- **Agent orchestration** — multi-agent workflows where one agent's output
  feeds another's input (e.g., accessibility checker feeds recommendations to
  a fixer).
- **Plugin system** — extensible middleware/interceptor chain for pre/post
  processing of analysis requests and results.
- **Observability integration** — OpenTelemetry spans for the analysis
  lifecycle (start, model call, finish).
- **Prompt templates** — pre-built, parameterized prompt templates for common
  use cases (accessibility audit, UX review, layout analysis, bug detection).
- **Diff analysis** — compare two screenshots and describe the differences
  structurally.
- **Video frame analysis** — extract frames from video and analyze them as a
  batch or sequence.

## API evolution

- **`Analyzer` interface expansion** — decide whether `AnalyzeConversation` and
  `AnalyzeStructured` belong on the `Analyzer` interface (breaking) or a
  separate `ConversationAnalyzer` interface (composable, non-breaking).
- **Remove deprecated `VisionAgent` alias** — `VisionAgent = Agent` exists for
  backwards compat; plan a removal timeline.
- **`Agent.Close`** — resource cleanup hook for agents that hold long-lived
  connections or state.
- **`Conversation.LastMessage` helper** — common access pattern.
- **`BatchResult.Duration`** — per-image analysis timing.

---

## Open questions

These need a product/architecture decision before they can become actionable
tasks. They are **not** TODO items until answered.

1. **catwalk or hand-rolled CLI providers?** The three providers added to
   `cmd/vision/main.go` (Anthropic, Google, openaicompat) work but will rot as
   fantasy evolves. Replace them with a `github.com/charmbracelet/catwalk`
   integration, or keep hand-rolled and layer catwalk on top?
2. **Retry strategy: bake in or keep external?** Should `RetryConfig` become a
   `Config` field (every `Analyze*` retries automatically, `Config.MaxRetries`
   goes away), or must `WithRetry[T]` stay an explicit per-call wrapper?
3. **Structured hooks payload: is a breaking API change acceptable?** Fixing
   the nil-`RawResponse` / JSON-as-`Text` hack properly means changing what
   `OnFinish` receives for structured calls. Is a breaking `Hooks` change OK
   in the next minor, or must `Hooks` stay stable?
4. **Semver policy for 0.x** — is 0.x "anything goes" or semver-lite? Should
   breaking changes get a `### Breaking` callout in CHANGELOG?
