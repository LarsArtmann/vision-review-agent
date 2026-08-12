# Roadmap

Long-term direction and raw ideas not yet refined into actionable tasks.
For short-term actionable work, see [TODO_LIST.md](TODO_LIST.md).
For current feature inventory, see [FEATURES.md](FEATURES.md).

> Items graduate from here to `TODO_LIST.md` when they become bounded and
> actionable. Items that shipped move to [CHANGELOG.md](CHANGELOG.md) and leave
> this file.

---

## Near-term direction

The actionable near-term work (preprocessing auto-wiring, retry reconciliation,
structured-hooks redesign, CLI tests, CostTracker integration, license fix,
config-hygiene cleanup) lives in [TODO_LIST.md](TODO_LIST.md). The themes:

- **Make preprocessing composable** — `ResizeImage` shipped standalone; it
  needs `Config.Preprocess` wiring so every `Analyze*` can auto-resize. Blocked
  on a design decision (see Open questions for the breaking-change question).
- **One retry system, not two** — `Config.MaxRetries` and `WithRetry[T]` must
  be reconciled. Blocked on a design decision (see Open questions).
- **catwalk integration for CLI** — replace hand-rolled CLI providers
  (Anthropic, Google, openaicompat) with `charmbracelet/catwalk` to avoid rot
  as fantasy adds providers. Blocked on a direction decision (see Open
  questions).

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
- **`Conversation.LastMessage` helper** — shipped in v0.4.0 (common access pattern).
- **`BatchResult.Duration`** — shipped in v0.4.0 (per-image analysis timing).

---

## Open questions

These need a product/architecture decision before they can become actionable
tasks. They are **not** TODO items until answered.

1. **Retry strategy: bake in or keep external?** Should `RetryConfig` become a
   `Config` field (every `Analyze*` retries automatically, `Config.MaxRetries`
   goes away), or must `WithRetry[T]` stay an explicit per-call wrapper?
2. **Structured hooks payload: is a breaking API change acceptable?** Fixing
   the nil-`RawResponse` / JSON-as-`Text` hack properly means changing what
   `OnFinish` receives for structured calls. Is a breaking `Hooks` change OK
   in the next minor, or must `Hooks` stay stable?
3. **Semver policy for 0.x** — is 0.x "anything goes" or semver-lite? Should
   breaking changes get a `### Breaking` callout in CHANGELOG?
4. **Tag anomaly resolution (partially done in `v0.4.0`; `v0.2.1` remains).**
   The tags `v0.2.1` and `v0.3.0` both pointed to commit `d5dda4b`
   (2026-04-27), an _ancestor_ of the real `v0.2.0` (`003a256`, 2026-07-23) and
   even older than `v0.1.0`. They never represented real releases. **Resolved
   for `v0.3.0`:** the bogus `v0.3.0` tag was deleted (local + remote) as part
   of the `v0.4.0` release, and the real post-v0.2.0 body of work ships as
   `v0.4.0`. A fresh `v0.3.0` was **not** recreated because the number is
   permanently burned on `proxy.golang.org` as `d5dda4b` (reusing it would
   cause checksum mismatches for anyone who resolved the ghost). **Still open:
   `v0.2.1`** is a ghost on the same commit `d5dda4b`. It has no functional
   impact (it sorts below `v0.2.0`/`v0.4.0` and blocks no number), but for a
   clean history it should be deleted too. Needs approval because deleting a
   remote tag is destructive and affects anyone who fetched it. Until then it
   stays documented, not acted on.
