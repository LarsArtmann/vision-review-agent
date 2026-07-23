# TODO List

Short- and mid-term actionable improvement tasks. Each item is bounded and scoped.
For long-term ideas and raw direction, see [ROADMAP.md](ROADMAP.md).
For current feature inventory, see [FEATURES.md](FEATURES.md).

---

## Bug Fixes

- [ ] **Wire hooks into `AnalyzeConversation`** — Hooks are silently ignored; users who set OnStart/OnFinish/OnError get no callbacks
- [ ] **Wire hooks into `AnalyzeConversationStream`** — Same gap
- [ ] **Wire hooks into `AnalyzeStructured`** — Same gap
- [ ] **Wire hooks into `AnalyzeStructuredStream`** — Same gap
- [ ] **Migrate `ScreenshotAnalyzer` errors to `classifyModelErr`** — Uses `wrapWithPrompt`, giving unclassified errors while `Agent` returns `*ModelError`. Users get different error types depending on which analyzer they use.
- [ ] **Remove dead `wrapWithPrompt` function** — Replaced by `classifyModelErr` at all Agent call sites; only ScreenshotAnalyzer still uses it (see item above)
- [ ] **Unify `AnalyzeConversationStream` validation** — Uses inline `if prompt == ""` + `requireImages` instead of the shared `validateAnalyzeInput` helper

## Missing Tests

- [ ] **Test ScreenshotAnalyzer `AnalyzeConversation` delegation** — Zero coverage
- [ ] **Test ScreenshotAnalyzer `AnalyzeConversationStream` delegation** — Zero coverage
- [ ] **Test `LoadImageFromURL` rejects non-image responses** — Currently accepts any binary data without magic byte validation
- [ ] **Fuzz test `decodeBase64Flex`** — Edge cases in base64 decoding paths
- [ ] **Fuzz test `DetectImageFormat`** — Edge cases in magic byte detection

## Missing Examples

- [ ] **`examples/conversation/`** — Multi-turn conversation usage
- [ ] **`examples/batch/`** — Concurrent batch analysis
- [ ] **`examples/hooks/`** — Lifecycle hooks for logging/metrics
- [ ] **`examples/structured-stream/`** — Structured streaming with partial objects
- [ ] **`examples/url-loading/`** — Load image from URL and base64

## API Consistency

- [ ] **Add `MediaTypeBMP` constant** — `DetectImageFormat` detects BMP but there's no typed constant in the `MediaType` enum
- [ ] **Replace `applyOptionalPointers` with direct field assignment** — The 6-parameter double-pointer helper is fragile; reordering args silently sets wrong fields
- [ ] **Add `ScreenshotAnalyzer.WithHooks`** — No way to set hooks via the fluent builder
- [ ] **Add `Conversation.Clear` method** — Reset history without creating a new instance
- [ ] **Document or fix `go.work` conflict** — `GOWORK=off` needed outside nix shell

## CLI Improvements

- [ ] **Add `-structured` flag to CLI** — Built-in UIReview schema with JSON output
- [ ] **Add Anthropic provider to CLI** — `anthropic` package exists in fantasy
- [ ] **Add Google provider to CLI** — `google` package exists in fantasy
- [ ] **Add `openaicompat` provider to CLI** — For local models (Ollama, LM Studio)
- [ ] **Run golangci-lint and address CLI warnings** — 39 lint warnings project-wide (forbidigo, exhaustruct, tagliatelle, mnd)

## Capability Exposure

- [ ] **Expose tool/function calling** — `Config.Tools`, wire to `fantasy.WithTools` and `NewAgentTool[T]`
- [ ] **Expose `ToolChoice`** — `Config.ToolChoice`, wire to `fantasy.WithToolChoice`
- [ ] **Expose `PrepareStep` interceptor** — Per-step model/prompt/tool manipulation
- [ ] **Expose stop conditions** — `Config.StopConditions` (StepCountIs, HasToolCall, MaxTokensUsed)
- [ ] **Expose `Headers` and `UserAgent` config** — fantasy supports both via `WithHeaders` / `WithUserAgent`
- [ ] **Add custom HTTP client to `LoadImageFromURL`** — Currently hardcodes `http.DefaultClient`

## Hardening

- [ ] **Validate image format in `LoadImageFromURL`** — Run `ValidateImage` after download to reject non-image responses
- [ ] **Add retry middleware with backoff** — Configurable, respects `IsRetryable()`
- [ ] **Add cost tracking** — Track input/output/cache/reasoning tokens across calls
- [ ] **Add image preprocessing** — Resize/compress before sending to model
