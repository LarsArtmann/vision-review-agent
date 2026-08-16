# Roadmap

Long-term direction and raw ideas not yet refined into actionable tasks.
For short-term actionable work, see [TODO_LIST.md](TODO_LIST.md).
For current feature inventory, see [FEATURES.md](FEATURES.md).

> Items graduate from here to `TODO_LIST.md` when they become bounded and
> actionable. Items that shipped move to [CHANGELOG.md](CHANGELOG.md) and leave
> this file.

---

## Near-term direction

The actionable near-term work is in [TODO_LIST.md](TODO_LIST.md): the **CI
lint-config fix** (red since v0.5.0), **visionreviewd activation** (SystemNix
input bump, first host, real-model bring-up), and the **tag anomaly**
(destructive, needs user approval). All previous near-term work — preprocessing
auto-wiring, retry reconciliation, catwalk CLI integration, cost tracking, the
visionreviewd daemon itself — has shipped and moved to
[CHANGELOG.md](CHANGELOG.md).

## Mid-term ideas

### visionreviewd operations

- **Retention/GC** — blob store and event journal grow unboundedly; add a
  policy and a `visionreviewd gc` (or documented manual pruning).
- **Daemon ergonomics** — SIGHUP config reload, `events -json` + pagination,
  `replay --dry-run` with progress, `visionreviewd tail` (live journal
  follow), per-project overrides (model, timeout, interval).
- **Review quality** — link archived blob images from comparison markdown,
  per-view comparison counts in INDEX, notify on score drops ≥ N.

### SDK (vision)

- **Provider-defined tools** — expose `WithProviderDefinedTools` for
  provider-native tools like web search and computer use (typed tools are
  already exposed via `Config.Tools`).
- **Context-aware batch** — let batch analysis share context or conversation
  state across images.
- **Batch-level hooks** — `Hooks.OnBatchStart` / `OnBatchFinish` for
  batch-scoped observability (per-image hooks already fire via internal
  `Analyze`).
- **Typed config-validation errors** — move beyond sentinel errors to typed
  validation failures that carry the field name, the invalid value, and the
  allowed range.
- **Custom HTTP client for providers** — `LoadImageFromURLWithClient` exists
  for image loading; extend the pattern to provider connections (proxies,
  timeouts, TLS).
- **Provider failover** — automatically try a secondary provider if the primary
  fails with a retryable error.
- **EXIF stripping** — strip EXIF metadata during preprocessing for privacy.
- **Remote-sync observability** — `Sync.Fetch` falls back silently; log
  fallback events, expose `LastFetchTime` / `IsStale`.

### Model catalog (catwalk)

- **Conveniences** — `ModelInfo.SupportsVision()`,
  `CostTracker.SetPricingFromModelInfo`, `catalog.Service.Stats()`,
  `FindVisionModel`, cached `VisionModels()`, `ModelInfo.Validate()`.
- **Quality surface** — BDD specs for catalog discovery and priced cost
  tracking; benchmarks for `FindModel`/`VisionModels()` across 800+ models;
  `examples/catalog` + `examples/cost-tracking` walkthroughs.

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

---

## Open questions

These need a product/architecture decision before they can become actionable
tasks. They are **not** TODO items until answered.

1. **Structured hooks payload: is a breaking API change acceptable?** Fixing
   the nil-`RawResponse` / JSON-as-`Text` hack properly means changing what
   `OnFinish` receives for structured calls. Is a breaking `Hooks` change OK
   in the next minor, or must `Hooks` stay stable?
2. **Semver policy for 0.x** — is 0.x "anything goes" or semver-lite? Should
   breaking changes get a `### Breaking` callout in CHANGELOG?
3. **Is `erraudit` a CI gate or an advisory tool?** Three error-system
   sessions left its 125 findings (mostly false-positive `context_loss` on
   result variables and anti-idiomatic `generic_return`) untouched because
   the answer is unknown. If it is a gate, suppression config is needed; if
   advisory, the documented false-positive rationale stands.
4. **Tag anomaly resolution (partially done in `v0.4.0`; both ghosts
   remain).** `v0.2.1` and `v0.3.0` both point at commit `d5dda4b`
   (2026-04-27), an _ancestor_ of the real `v0.2.0` (`003a256`, 2026-07-23).
   They never represented real releases. **Correction 2026-08-16:** the
   v0.4.0 release note claims `v0.3.0` was deleted (local + remote), but
   both tags currently exist on `origin` — the deletion either never took or
   was undone. A fresh `v0.3.0` must NOT be created regardless: the number
   is burned on `proxy.golang.org` as `d5dda4b` (reusing it would cause
   checksum mismatches). Deleting both ghosts needs approval because remote
   tag deletion is destructive. Until then they stay documented, not acted
   on (see TODO_LIST "Release mechanics").
