# TODO List

Short- and mid-term actionable improvement tasks. Each item is bounded and scoped.
For long-term ideas and raw direction, see [ROADMAP.md](ROADMAP.md).
For current feature inventory, see [FEATURES.md](FEATURES.md).

---

> **Status (2026-07-27):** All items below were completed in a single
> comprehensive pass. Each is verified by passing tests (`go test ./...`),
> `go vet`, and `golangci-lint run` (0 issues). New features are documented in
> [FEATURES.md](FEATURES.md).

## Bug Fixes

- [x] **Wire hooks into `AnalyzeConversation`** — OnStart/OnFinish/OnError now fire
- [x] **Wire hooks into `AnalyzeConversationStream`** — Same; validation unified via `validateAnalyzeInput`
- [x] **Wire hooks into `AnalyzeStructured`** — Same
- [x] **Wire hooks into `AnalyzeStructuredStream`** — Same
- [x] **Migrate `ScreenshotAnalyzer` errors** — `wrapWithPrompt` removed; config errors returned directly (model errors already classified via delegation)
- [x] **Remove dead `wrapWithPrompt` function** — Deleted from `vision.go`
- [x] **Unify `AnalyzeConversationStream` validation** — Now uses shared `validateAnalyzeInput`

## Missing Tests

- [x] **ScreenshotAnalyzer `AnalyzeConversation` delegation** — BDD coverage added
- [x] **ScreenshotAnalyzer `AnalyzeConversationStream` delegation** — BDD coverage added
- [x] **`LoadImageFromURL` rejects non-image responses** — Test + `ValidateImage` enforcement
- [x] **Fuzz test `decodeBase64Flex`** — Round-trip + stability invariants
- [x] **Fuzz test `DetectImageFormat`** — Consistency with `IsValidImage`, no unknown formats

## Missing Examples

- [x] **`examples/conversation/`** — Multi-turn conversation usage
- [x] **`examples/batch/`** — Concurrent batch analysis
- [x] **`examples/hooks/`** — Lifecycle hooks for logging/metrics
- [x] **`examples/structured-stream/`** — Structured streaming with partial objects
- [x] **`examples/url-loading/`** — Load image from URL (custom client) and base64

## API Consistency

- [x] **Add `MediaTypeBMP` constant** — Added to the `MediaType` enum
- [x] **Replace `applyOptionalPointers` with direct field assignment** — Inlined in both call sites
- [x] **Add `ScreenshotAnalyzer.WithHooks`** — Fluent builder method + cache invalidation
- [x] **Add `Conversation.Clear` method** — Reset history, returns same instance
- [x] **Document `go.work` conflict** — No `go.work` in repo; `GOWORK=off` documented in AGENTS.md

## CLI Improvements

- [x] **Add `-structured` flag to CLI** — Built-in `uiReview` schema with JSON output
- [x] **Add Anthropic provider to CLI** — `ANTHROPIC_API_KEY`
- [x] **Add Google provider to CLI** — Application Default Credentials
- [x] **Add `openaicompat` provider to CLI** — `OPENAICOMPAT_BASE_URL` (+ optional key)
- [x] **Run golangci-lint and address warnings** — 0 issues (was 39 project-wide)

## Capability Exposure

- [x] **Expose tool/function calling** — `Config.Tools` → `fantasy.WithTools`
- [x] **Expose `ToolChoice`** — `Config.ToolChoice` → `fantasy.WithToolChoice`
- [x] **Expose `PrepareStep` interceptor** — `Config.PrepareStep` → `fantasy.WithPrepareStep`
- [x] **Expose stop conditions** — `Config.StopConditions` → `fantasy.WithStopConditions`
- [x] **Expose `Headers` and `UserAgent` config** — Wired to `fantasy.WithHeaders`/`WithUserAgent`
- [x] **Add custom HTTP client to `LoadImageFromURL`** — `LoadImageFromURLWithClient`

## Hardening

- [x] **Validate image format in `LoadImageFromURL`** — Runs `ValidateImage` after download
- [x] **Add retry middleware with backoff** — `WithRetry[T]` + `RetryConfig` (respects `IsRetryable()`)
- [x] **Add cost tracking** — `CostTracker` accumulates `fantasy.Usage` across calls
- [x] **Add image preprocessing** — `ResizeImage` (Catmull-Rom, aspect-preserving)
