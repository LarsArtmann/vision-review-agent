# Status Report: 2026-07-23 16:08 — Post-Feature-Build Brutal Self-Review

---

## Executive Summary

This session took the vision-review-agent SDK from a basic single-shot image analysis library to a substantially richer platform: multi-turn conversations, batch processing, structured streaming, lifecycle hooks, flexible image loading, full model parameters, classified errors, and a cache bug fix. Build passes, vet is clean, race-detector tests pass, coverage is 84.5% / 95.9%.

**However**, this was executed alongside a parallel agent process that committed overlapping work (error classification model, hooks wiring, batch file). The result is a codebase that works but has rough edges, gaps, and inconsistencies that a top-tier engineer would not ship.

---

## A) FULLY DONE (high confidence, tested, verified)

| #   | Feature                                                          | Files                                                 | Evidence                                                                              |
| --- | ---------------------------------------------------------------- | ----------------------------------------------------- | ------------------------------------------------------------------------------------- |
| 1   | `LoadImageFromURL`                                               | `image.go`                                            | HTTP download with Content-Type detection, context-aware, tested with httptest server |
| 2   | `LoadImageFromBase64`                                            | `image.go`                                            | Standard/URL-safe/raw base64 decoding, tested with all 3 encodings + error cases      |
| 3   | `Conversation` type                                              | `conversation.go`                                     | AddUserMessage/AddAssistantMessage, nil filtering, tested                             |
| 4   | `AnalyzeConversation` + Stream                                   | `vision.go`                                           | Both on Agent and ScreenshotAnalyzer, tested with mock                                |
| 5   | `AnalyzeBatch`                                                   | `batch.go`                                            | Concurrent with semaphore, per-image error capture, nil handling, tested              |
| 6   | `Hooks` (OnStart/OnFinish/OnError)                               | `hooks.go`                                            | Wired into Analyze + AnalyzeStream, tested with atomic counter                        |
| 7   | Config expansion (TopP, TopK, PresencePenalty, FrequencyPenalty) | `vision.go`, `errors.go`                              | Validation, wiring to fantasy options, per-call overrides, tested                     |
| 8   | ScreenshotAnalyzer cache invalidation fix                        | `screenshot.go`                                       | All 9 builder methods set cachedAgent=nil, tested                                     |
| 9   | `AnalyzeStructuredStream[T]`                                     | `structured.go`                                       | Streams partial objects via callback, tested with mock                                |
| 10  | AnalyzeStructured param fix                                      | `structured.go`                                       | Now passes all 6 model params to ObjectCall                                           |
| 11  | Documentation updates                                            | `README.md`, `FEATURES.md`, `ROADMAP.md`, `AGENTS.md` | All new features documented                                                           |

---

## B) PARTIALLY DONE (shipped but incomplete or with gaps)

### B1. Hooks NOT wired into AnalyzeConversation / AnalyzeConversationStream

`Hooks` are wired into `Analyze` and `AnalyzeStream` but **NOT** into `AnalyzeConversation` or `AnalyzeConversationStream`. A user who sets up hooks and uses conversation methods will get **silently ignored hooks**. This is inconsistent and surprising.

**Severity**: Medium — functional gap that breaks the principle of least surprise.

### B2. Hooks NOT wired into AnalyzeStructured / AnalyzeStructuredStream

Same gap. Structured output methods don't fire hooks either.

**Severity**: Medium.

### B3. AnalyzeConversationStream uses inconsistent validation

`AnalyzeConversation` uses `validateAnalyzeInput` (the centralized helper), but `AnalyzeConversationStream` uses inline `if prompt == ""` + `requireImages` checks. This is an inconsistency from the parallel-agent merge — the methods were written at different times.

**Severity**: Low — works correctly but violates DRY.

### B4. ScreenshotAnalyzer conversation delegation NOT tested

I added `AnalyzeConversation` and `AnalyzeConversationStream` delegation methods to `ScreenshotAnalyzer` but there are **no tests** for them. The methods compile and delegate correctly, but there's zero test coverage.

**Severity**: Medium — untested code is broken code.

### B5. `applyOptionalPointers` is overly clever

The parallel agent introduced `applyOptionalPointers` with 6 `**` pointer parameters in a specific canonical order. This is fragile — if someone reorders the arguments at a call site, it silently sets the wrong field. The comment says "Callers that need a single shared helper across two external structs" but the two structs (`AgentCall` and `AgentStreamCall`) have identical field types, so this optimization saves ~20 lines at the cost of readability and safety.

**Severity**: Low — works but violates "make wrong code hard to write."

### B6. Examples not updated

No new examples for conversation, batch, hooks, URL/base64 loading, or structured streaming. The FEATURES.md notes this as "PARTIALLY DONE" but it means a new user has no copy-paste starting point for the most exciting new features.

**Severity**: Medium — examples are the first thing users look at.

### B7. Error classification tests reference `errors.AsType` (Go 1.26+ generic)

The test file `error_classification_test.go` uses `errors.AsType[*apperrors.ModelError](err)` which is the Go 1.26+ generic pattern. This is correct for the project's `go 1.26.4` requirement, but the `AGENTS.md` skill on hierarchical-errors exists and was **never consulted**. We may be missing linter-recommended patterns.

**Severity**: Low.

---

## C) NOT STARTED (planned but never began)

### C1. Tool/function calling exposure

fantasy supports `WithTools`, `NewAgentTool[T]`, `WithToolChoice`, and provider-defined tools. None of this is exposed through the vision SDK. This was on the original plan but was deferred.

### C2. PrepareStep interceptor exposure

fantasy's `WithPrepareStep` allows per-step model/system/tool manipulation. Not exposed.

### C3. Stop conditions

fantasy's composable stop conditions (`StepCountIs`, `HasToolCall`, `MaxTokensUsed`). Not exposed.

### C4. Additional CLI providers

CLI only supports OpenAI and OpenRouter. fantasy supports Anthropic, Google, Azure, Bedrock, Vercel, openaicompat, and kronk. None wired.

### C5. Retry middleware with backoff

ROADMAP.md mentions it, no work started.

### C6. Cost tracking

ROADMAP.md mentions it, no work started.

### C7. Image preprocessing (resize/compress)

ROADMAP.md mentions it, no work started.

### C8. Custom HTTP client injection

`LoadImageFromURL` uses `http.DefaultClient`. No way to inject a custom client.

---

## D) TOTALLY FUCKED UP (nothing catastrophic, but these are real problems)

### D1. Parallel agent collision

A parallel agent process committed work (commits b3a83b4 through 9cab7d3) while this session was running. This caused:

- The `structured.go` file was modified out from under me mid-edit (got "file modified since last read" error)
- The `withTimeout` function was temporarily broken when my edit collided
- Error handling code (`classifyModelErr`, `apperrors.Wrap`, `apperrors.KindStructuredParse`) appeared that I didn't write and had to reverse-engineer
- The mock_test.go was modified by the other process to add error injection fields

**Impact**: No lasting damage (everything compiles and tests pass), but I lost time and had to re-read files multiple times. The merge of two agents' work created the hook-wiring gaps (B1/B2) because each agent wired hooks into different methods.

### D2. `go.work` conflict

Running `go test ./...` fails with "directory prefix . does not contain modules listed in go.work" without `GOWORK=off`. The `flake.nix` sets `GOWORK = "off"` in the dev shell, but any command run outside the nix shell hits this. This is a pre-existing issue, not caused by this session, but it's annoying and should be documented or fixed.

---

## E) WHAT WE SHOULD IMPROVE (things I noticed but didn't fix)

### E1. Wire hooks into ALL analysis methods

Hooks are only in `Analyze` and `AnalyzeStream`. They need to be in `AnalyzeConversation`, `AnalyzeConversationStream`, `AnalyzeStructured`, and `AnalyzeStructuredStream` too. This is the single most important fix.

### E2. Unify validation in conversation methods

`AnalyzeConversationStream` should use `validateAnalyzeInput` like its siblings.

### E3. Replace `applyOptionalPointers` with direct field assignment

The 6-parameter double-pointer helper is too clever. Direct assignment in `applyModelParamsAgentCall` and `applyModelParamsStreamCall` (like the original code did) is clearer and equally DRY.

### E4. Add ScreenshotAnalyzer conversation tests

Zero coverage on the delegation methods.

### E5. Add examples for new features

Conversation, batch, hooks, URL/base64, structured streaming.

### E6. CLI should support structured output

The CLI only does text analysis. Adding `-structured` flag with a built-in UIReview schema would showcase the structured output capability.

### E7. Consider `MediaTypeBMP` constant

`DetectImageFormat` detects BMP, but there's no `MediaTypeBMP` constant in the `MediaType` enum. This is an inconsistency.

### E8. `LoadImageFromURL` should validate image format

After downloading, the data is not validated against magic bytes. A URL could return arbitrary binary data.

### E9. Batch analysis doesn't use hooks

`AnalyzeBatch` calls `Analyze` internally, so hooks fire per-image, but there's no batch-level hook (OnBatchStart/OnBatchFinish). This may be fine, but it's worth deciding intentionally.

### E10. `wrapWithPrompt` is now dead code

The parallel agent replaced `wrapWithPrompt` with `classifyModelErr` at most call sites, but `wrapWithPrompt` still exists in `vision.go:449`. If it's truly unused, it should be removed. If it's used by `ScreenshotAnalyzer` delegation, those should be migrated too.

### E11. ScreenshotAnalyzer `wrapWithPrompt` inconsistency

`ScreenshotAnalyzer` methods use `wrapWithPrompt` for errors, while `Agent` methods use `classifyModelErr`. Users get different error types depending on which analyzer they use.

---

## F) Up to 50 Things We Should Get Done Next

### High Priority (breaks consistency or correctness)

1. **Wire hooks into `AnalyzeConversation`** — Currently silently ignored
2. **Wire hooks into `AnalyzeConversationStream`** — Currently silently ignored
3. **Wire hooks into `AnalyzeStructured`** — Currently silently ignored
4. **Wire hooks into `AnalyzeStructuredStream`** — Currently silently ignored
5. **Unify `AnalyzeConversationStream` validation** — Use `validateAnalyzeInput` instead of inline checks
6. **Test ScreenshotAnalyzer conversation delegation** — Zero coverage
7. **Remove or migrate `wrapWithPrompt`** — Dead code or inconsistency
8. **Migrate ScreenshotAnalyzer errors to `classifyModelErr`** — Currently uses `wrapWithPrompt`, giving unclassified errors

### Medium Priority (quality and completeness)

9. **Add `MediaTypeBMP` constant** — DetectImageFormat detects it but no typed constant
10. **Validate image format in `LoadImageFromURL`** — Currently accepts any binary data
11. **Add conversation example** — `examples/conversation/main.go`
12. **Add batch example** — `examples/batch/main.go`
13. **Add hooks example** — `examples/hooks/main.go`
14. **Add structured streaming example** — `examples/structured-stream/main.go`
15. **Add URL loading example** — `examples/url-loading/main.go`
16. **Expose tool/function calling** — `Config.Tools []AgentTool`, wire to `fantasy.WithTools`
17. **Expose `ToolChoice`** — `Config.ToolChoice`, wire to `fantasy.WithToolChoice`
18. **Expose `PrepareStep` interceptor** — For per-step model/prompt/tool manipulation
19. **Expose stop conditions** — `Config.StopConditions`, wire to `fantasy.WithStopConditions`
20. **Add Anthropic provider to CLI** — `anthropic` package exists in fantasy
21. **Add Google provider to CLI** — `google` package exists in fantasy
22. **Add Azure provider to CLI** — `azure` package exists in fantasy
23. **Add `openaicompat` provider to CLI** — For local models (Ollama, LM Studio)
24. **Add `-structured` flag to CLI** — Built-in UIReview schema, JSON output
25. **Replace `applyOptionalPointers` with direct assignment** — Too clever, fragile
26. **Document or fix `go.work` issue** — GOWORK=off needed outside nix shell
27. **Add `LoadImageFromURL` format validation test** — Verify it rejects non-image responses
28. **Add `LoadImageFromURL` custom HTTP client option** — For proxies, timeouts, TLS

### Lower Priority (polish and future direction)

29. **Add retry middleware with exponential backoff** — Configurable, respects `IsRetryable()`
30. **Add cost tracking** — Track input/output/cache/reasoning tokens across calls
31. **Add image preprocessing** — Resize, compress before sending to model
32. **Add result caching by image hash** — Avoid redundant API calls
33. **Add provider failover** — Try secondary provider on retryable failure
34. **Add OpenTelemetry spans** — For analysis lifecycle observability
35. **Add prompt templates** — Pre-built parameterized prompts (accessibility, UX review, etc.)
36. **Add diff analysis** — Compare two screenshots, describe differences structurally
37. **Add `ScreenshotAnalyzer.WithHooks`** — Currently no way to set hooks on the builder
38. **Add `ScreenshotAnalyzer.WithTopP` test in BDD suite** — Builder methods untested in BDD
39. **Add batch-level hooks** — `Hooks.OnBatchStart` / `Hooks.OnBatchFinish`
40. **Run golangci-lint and fix all warnings** — 39 lint warnings in project (pre-existing + new)
41. **Add fuzz tests for base64 decoding** — Edge cases in `decodeBase64Flex`
42. **Add fuzz tests for image format detection** — Edge cases in `DetectImageFormat`
43. **Add integration test with real httptest server** — End-to-end analysis flow
44. **Add `Conversation.Clear` method** — Reset history without creating new instance
45. **Add `Conversation.LastMessage` helper** — Common pattern
46. **Add `BatchResult.Duration` field** — Track per-image analysis time
47. **Add `Config.Headers` field** — Custom HTTP headers (fantasy supports this)
48. **Add `Config.UserAgent` field** — Override User-Agent header (fantasy supports this)
49. **Consider `Analyzer` interface expansion** — Add `AnalyzeConversation` to the interface
50. **Write CONTRIBUTING.md update** — Document new feature contribution patterns

---

## G) Questions I CANNOT Answer Myself

### Q1: Should `AnalyzeConversation` and `AnalyzeStructured` be on the `Analyzer` interface?

Currently `Analyzer` only declares `Analyze` and `AnalyzeStream`. Adding `AnalyzeConversation` would force every implementor (including consumer mocks) to implement it. Expanding the interface is a breaking change. Should we:

- (a) Expand `Analyzer` (breaking change, but honest interface)
- (b) Create a separate `ConversationAnalyzer` interface (composable, non-breaking)
- (c) Leave them as concrete methods on `*Agent` only (current state, but limits mockability)

### Q2: Should we keep the `VisionAgent` type alias?

`VisionAgent = Agent` exists as a deprecated backwards-compat alias. At what point do we remove it? Go has no formal deprecation cycle. Removing it is a breaking change. Keeping it clutters the API surface.

### Q3: Should `LoadImageFromURL` use a configurable HTTP client, or should we add a separate `LoadImageFromURLWithClient` function?

Adding a `client *http.Client` parameter to `LoadImageFromURL` changes the signature (breaking). Adding a separate function avoids the break but adds API surface. A functional-options pattern (`LoadImageFromURL(ctx, url, opts...)`) is the most flexible but adds complexity for a single optional parameter.

---

## Resolution (2026-07-27)

Re-verified against code four days later. The bulk of sections B / C / E / F
shipped in the 2026-07-27 TODO_LIST execution pass (see
`2026-07-27_11-49_post-todo-execution-brutal-review.md`). Three items shipped
**but flawed** — see the middle group.

### Shipped cleanly (DONE)

| Item | Claim in report | Resolution | Evidence |
| ---- | --------------- | ---------- | -------- |
| B1 | Hooks not wired into `AnalyzeConversation`/`Stream` | **DONE** | `TestHooksFireAcrossAllAnalysisMethods` |
| B3 | `AnalyzeConversationStream` inline validation | **DONE** — uses `validateAnalyzeInput` | code review |
| B4 | ScreenshotAnalyzer conversation delegation untested | **DONE** — BDD "Conversation Delegation" (3 specs) | `screenshot_bdd_test.go` |
| B5 / F.25 | `applyOptionalPointers` too clever | **DONE** — removed; direct field assignment | `vision.go` |
| B6 / F.11–F.15 | No examples for new features | **DONE** — 5 examples added (conversation, batch, hooks, structured-stream, url-loading) | `examples/` |
| C1 / F.16 | Tool/function calling exposure | **DONE** — `Config.Tools` → `fantasy.WithTools` | `vision.go` |
| C2 / F.18 | `PrepareStep` interceptor | **DONE** — `Config.PrepareStep` | `vision.go` |
| C3 / F.19 | Stop conditions | **DONE** — `Config.StopConditions` | `vision.go` |
| C4 / F.20–F.23 | Additional CLI providers | **DONE** — Anthropic, Google (ADC), openaicompat added | `cmd/vision/main.go` |
| C8 / F.28 / Q3 | Custom HTTP client for `LoadImageFromURL` | **DONE** — chose separate `LoadImageFromURLWithClient` fn | `image.go` |
| E2 / F.5 | Unify `AnalyzeConversationStream` validation | **DONE** | as B3 |
| E4 / F.6 | ScreenshotAnalyzer conversation tests | **DONE** | as B4 |
| E5 | Add examples | **DONE** | as B6 |
| E6 / F.24 | CLI structured output | **DONE** — `-structured` flag + built-in `uiReview` schema | `cmd/vision/main.go` |
| E7 / F.9 | `MediaTypeBMP` constant | **DONE** — added to enum (but see "still open" for decoder + ext detection) | `image.go:24` |
| E8 / F.10 / F.27 | Validate image format in `LoadImageFromURL` | **DONE** — runs `ValidateImage` post-download | `image.go` |
| E10 / F.7 | Remove dead `wrapWithPrompt` | **DONE** — deleted repo-wide | `grep` → 0 |
| E11 / F.8 | Migrate ScreenshotAnalyzer errors to `classifyModelErr` | **DONE** — `wrapWithPrompt` gone; config errors returned directly | `screenshot.go` |
| F.1–F.4 | Wire hooks into all four analysis methods | **DONE** | `TestHooksFireAcrossAllAnalysisMethods` |
| F.17 | `ToolChoice` | **DONE** — `Config.ToolChoice` | `vision.go` |
| F.33 | `Conversation.Clear` | **DONE** | `conversation.go` |
| F.37 | `ScreenshotAnalyzer.WithHooks` | **DONE** — fluent builder + cache invalidation | `screenshot.go` |
| F.40 | golangci-lint cleanup (39 warnings) | **DONE** — 0 issues | `.golangci.yaml` |
| F.41 / F.42 | Fuzz tests for base64 + image format detection | **DONE** — `FuzzDecodeBase64Flex`, `FuzzDetectImageFormat` | `pkg/vision/` |
| F.47 / F.48 | `Config.Headers` / `Config.UserAgent` | **DONE** — wired to fantasy | `vision.go` |

### Shipped but flawed (PARTIALLY DONE)

| Item | Claim in report | Resolution | Evidence |
| ---- | --------------- | ---------- | -------- |
| B2 / E1 / F.3–F.4 | Hooks not wired into `AnalyzeStructured`/`Stream` | **DONE but flawed** — hooks fire, but `fireFinish` receives a synthesized `&AnalyzeResult{Text, Usage}` with **nil `RawResponse`** (nil-pointer hazard) and `Text` holds raw JSON, not prose | `pkg/vision/structured.go:106,226`; tracked in TODO_LIST |
| C5 / F.29 | Retry middleware with backoff | **DONE but disjoint** — `WithRetry[T]` + `RetryConfig` shipped, but **conflicts with `Config.MaxRetries`** (two retry systems); not wired into `AnalyzeBatch`/`AnalyzeConversation` | `pkg/vision/retry.go`; tracked in TODO_LIST |
| C6 / F.30 | Cost tracking | **DONE but detached** — `CostTracker` shipped but no `Agent` integration; caller must wire `Hooks.OnFinish` manually | `pkg/vision/cost.go`; tracked in TODO_LIST |
| C7 / F.31 | Image preprocessing | **DONE but island** — `ResizeImage` shipped (Catmull-Rom) but never called by Agent; no `Config.Preprocess`; no compress/convert; BMP decode fails | `pkg/vision/preprocess.go`; tracked in TODO_LIST |
| C4 (runtime) | CLI providers Anthropic/Google/openaicompat | **BUILD-VERIFIED ONLY** — compile + `-h` pass, no credentials to runtime-test | `cmd/vision/main.go` |

### Still open

| Item | Status | Note |
| ---- | ------ | ---- |
| E9 / F.39 | Batch-level hooks (`OnBatchStart`/`OnBatchFinish`) | not added; per-image hooks fire via internal `Analyze` |
| F.32 | Result caching by image hash | not started |
| F.34 | Provider failover | not started |
| F.35 | OpenTelemetry spans | not started |
| F.36 | Prompt templates | not started |
| F.38 | `ScreenshotAnalyzer.WithTopP` BDD test | BDD coverage expanded generally; this specific builder test not confirmed |
| F.43 | Integration test with real httptest server | not started |
| F.45 | `Conversation.LastMessage` helper | not added |
| F.46 | `BatchResult.Duration` field | not added |
| Q1 | Expand `Analyzer` interface with `AnalyzeConversation`? | **unresolved** — still concrete methods on `*Agent` only |
| Q2 | Remove deprecated `VisionAgent` alias? | **unresolved** — still present |

**Net:** ~30 of the 50 follow-ups shipped; 5 shipped-but-flawed (the retry /
cost / preprocess / structured-hooks cluster); the rest remain open and are
tracked in `TODO_LIST.md` / `ROADMAP.md`.
