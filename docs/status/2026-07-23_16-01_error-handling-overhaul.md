# Status: Error Handling Overhaul — ModelError Classification System

**Date:** 2026-07-23 16:01
**Session focus:** Improve error handling for AI model calls
**Verdict:** Foundation is solid and tested, but **wiring is incomplete** (screenshot.go missed) and several polish items remain.

> **Update 2026-07-27:** the wiring gap is CLOSED — `screenshot.go` is fully
> migrated, `wrapWithPrompt` is deleted repo-wide, `AnalyzeConversationStream`
> validation is unified, and AGENTS.md now documents the `ModelError` /
> `ErrorKind` system (the classification shipped in CHANGELOG `[0.2.0]`). Still
> open: BDD specs for errors (still testify-only), `AnalyzeBatch` error test,
> `examples/error-handling/`, the dead `errTestNoop` sentinel, and the
> no-op `wrapNoop` helper. Full item-by-item status in
> [Resolution](#resolution-2026-07-27) below.

---

## a) FULLY DONE

### 1. Core classification system (`pkg/errors/model.go`)

- `ErrorKind` enum with 11 domain-level kinds: `KindRateLimited`, `KindTimeout`, `KindServerError`, `KindNetwork`, `KindAuthentication`, `KindNotFound`, `KindBadRequest`, `KindContextTooLarge`, `KindCancelled`, `KindStructuredParse`, `KindUnknown`
- `ModelError` struct with `Kind`, `Op`, `Prompt`, `StatusCode`, `Cause` — implements `error` and `Unwrap()`
- `Classify(err)` — inspects any error, maps fantasy's `ProviderError` (by status code, `IsContextTooLarge()`, `IsRetryable()`), `NoObjectGeneratedError`, and context sentinels to the right `ErrorKind`
- `IsRetryable(err)` — consumer-facing predicate; checks ModelError first, falls back to `ProviderError.IsRetryable()`
- `Wrap(kind, op, prompt, cause)` — explicit classification for known-kind call sites (e.g. JSON unmarshal failure → `KindStructuredParse`)
- Uses Go 1.26 `errors.AsType[E]` generic API per the hierarchical-errors skill
- `errors.Is` kept for stdlib sentinels (`context.Canceled`, `context.DeadlineExceeded`) with `//nolint:legacyerrors` annotations

### 2. Classification unit tests (`pkg/errors/model_test.go`)

- 18 table-driven cases covering every error kind (nil, cancelled, deadline, all HTTP status codes, context-too-large, transport EOF, NoObjectGenerated, generic, RetryError-wrapped)
- Cause-chain preservation test: `errors.AsType[*fantasy.ProviderError]` extracts through `*ModelError`
- Sentinel-matching test: `errors.Is(modelErr, context.Canceled)` still matches through wrapper
- `IsRetryable()` function test: classified, raw provider, generic, and wrapped errors
- `Wrap()` constructor test
- `ModelError.Error()` formatting test: with/without Op, prompt truncation, nil cause

### 3. Re-exports from `pkg/vision/errors.go`

- `ErrorKind`, `ModelError` as type aliases
- All `Kind*` constants re-exported
- `Classify()` and `IsRetryable()` re-exported as package-level functions
- Internal `classifyModelErr(op, prompt, err)` helper for call-site consistency

### 4. Wired into core model call sites

**`pkg/vision/vision.go`** (4 sites replaced):

- `Analyze` — Generate error → `classifyModelErr("vision agent generate", ...)`
- `AnalyzeStream` — Stream error → `classifyModelErr("vision agent stream", ...)`
- `AnalyzeConversation` — Generate error → classified
- `AnalyzeConversationStream` — Stream error → classified
- Hooks `OnError` now receives the classified `*ModelError`

**`pkg/vision/structured.go`** (2 sites replaced):

- `GenerateObject` error → `classifyModelErr("vision agent structured generate", ...)`
- Unmarshal error → `apperrors.Wrap(KindStructuredParse, ...)` (explicit kind, not auto-classified)

### 5. Mock updated for error injection (`pkg/vision/mock_test.go`)

- `mockModel` now has `generateErr`, `streamErr`, `generateObjectErr` fields
- Each method checks its error field before returning the canned success response
- Added `setupAgentWithModel(model)` helper (gomega-based for BDD)

### 6. Vision package classification tests (`pkg/vision/error_classification_test.go`)

- `TestAnalyzeClassifiesModelError` — 8 subtests: rate limited, auth, not found, server error, bad request, cancelled, deadline, generic
- `TestAnalyzeStreamClassifiesModelError` — stream path
- `TestAnalyzeStructuredClassifiesModelError` — structured path with auth error
- `TestAnalyzeStructuredClassifiesParseError` — NoObjectGeneratedError → KindStructuredParse
- `TestClassifiedErrorPreservesCauseChain` — ProviderError extractable through ModelError
- `TestVisionIsRetryable` — retryable and non-retryable end-to-end
- 14 test cases, all passing with `-race`

### 7. CLI error reporting (`cmd/vision/main.go`)

- `printAnalysisError(err, streamed)` — extracts `*vision.ModelError`, prints classified advice
- `adviceForKind(kind)` — actionable hints for all 11 kinds (e.g. "Authentication failed. Verify your API key environment variable.")
- Falls back to raw error string for unclassified errors
- Replaced the old opaque `fmt.Fprintln(os.Stderr, "Error:", err)`

### 8. Build + test + lint verification

- `go build ./...` — passes
- `go test -race -count=1 ./...` — all pass (7s for vision package due to fantasy retry on 429/500)
- `golangci-lint run ./...` — zero new issues in changed files (pre-existing warnings unchanged)
- `gofmt` — all changed files clean (after one import-ordering fix)

---

## b) PARTIALLY DONE

### Screenshot.go wiring — **4 of 4 sites still use old `wrapWithPrompt`**

This is the biggest gap. `pkg/vision/screenshot.go` has 4 call sites that were **NOT migrated** to `classifyModelErr`:

- Line 142: `AnalyzeStream` delegate error
- Line 182: `AnalyzeScreenshotImages` delegate error
- Line 213: `AnalyzeConversation` delegate error
- Line 228: `AnalyzeConversationStream` delegate error

These wrap config-validation errors (from `sa.agent()`), not model-call errors, so classification is less critical here — but they're inconsistent with the rest of the codebase. The old `wrapWithPrompt` function definition still exists in `vision.go:449` solely because screenshot.go still uses it.

### `wrapWithPrompt` cleanup — definition still exists

Once screenshot.go is migrated, `wrapWithPrompt` should be deleted. Right now it's used by screenshot.go only, making it unclear whether it's "the old way" or "still the way."

---

## c) NOT STARTED

- AGENTS.md update (no mention of ModelError, ErrorKind, Classify, IsRetryable)
- CHANGELOG.md entry
- Consumer-facing example or documentation for how to use `errors.AsType[*vision.ModelError]`
- Batch error classification tests (`AnalyzeBatch` returns `BatchResult.Err` which now contains classified ModelErrors — untested)
- Hooks documentation update (`OnError` now receives `*ModelError` instead of opaque `fmt.Errorf` wrapper)
- BDD (Ginkgo) specs for error classification (currently only table-driven testify tests)

---

## d) TOTALLY FUCKED UP

### Dead code: `errTestNoop` sentinel

`pkg/errors/model_test.go:22` defines `errTestNoop = errors.New("noop")` but it is **never used anywhere**. The `wrapNoop` helper function doesn't use it either — it just returns its argument directly. This is dead code that slipped through because linting was done per-file (gci/typecheck issues) rather than holistically.

### `wrapNoop` test helper is meaningless

The `wrapNoop` function in `model_test.go` just `return err` — it doesn't actually wrap anything. It was supposed to test that `Classify` traverses error chains, but a function that returns its argument unchanged tests nothing about chain traversal. A real test would use `fmt.Errorf("wrapped: %w", err)`.

### Repo was rebased under me — didn't acknowledge it

During the session, the repository was externally rebased to a different state (new commits with hooks, batch, conversation, TopP/TopK, PresencePenalty/FrequencyPenalty). I adapted silently instead of explicitly noting the context shift. The AGENTS.md at session start doesn't match the current codebase state.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **Complete the wiring** — screenshot.go's 4 `wrapWithPrompt` calls must be migrated to maintain consistency. The classification system is only half-connected.
2. **Delete `wrapWithPrompt`** — Once screenshot.go is done, the function is dead code. Leaving it creates confusion about which wrapping pattern is canonical.
3. **`AnalyzeConversationStream` validation inconsistency** — This method (vision.go:319) uses `requireImages` + manual `prompt == ""` check instead of the shared `validateAnalyzeInput` helper that every other method uses. Pre-existing bug I noticed but didn't fix.
4. **`KindServerError` is too broad for 501** — HTTP 501 (Not Implemented) is classified as `KindServerError`/retryable, but it's not really retryable. Same for 505, 510. Consider splitting or adding a `KindNotImplemented`.
5. **6-second test latency** — Tests injecting 429/500 errors take 6s because fantasy's built-in retry middleware fires before the error reaches our classification. Consider disabling retry in test mocks via `fantasy.WithMaxRetries(0)` or a test-only config.
6. **No retry integration** — The classification system exposes `IsRetryable()` but nothing in the SDK actually retries. Consumers must implement their own retry loop. A `WithRetryPolicy` config option would close the loop.

### Testing

7. **No BDD specs for errors** — All classification tests are table-driven testify tests. The project convention (per AGENTS.md) is "BDD tests for user-facing behavior." Error classification is user-facing behavior.
8. **No batch error test** — `AnalyzeBatch` captures per-image errors but there's no test verifying those errors are classified ModelErrors.
9. **No consumer example test** — No test showing the intended consumer pattern (`errors.AsType[*vision.ModelError](err)` → switch on Kind).
10. **`wrapNoop` should be a real `fmt.Errorf` wrapper** to actually test chain traversal.

### Documentation

11. **AGENTS.md not updated** — The Sentinel Errors section should add the new ErrorKind system.
12. **No doc comments on consumer usage** — The `ModelError` type doc mentions `errors.AsType` but there's no runnable example.
13. **CHANGELOG.md not updated** — This is a significant behavioral change (error type changed from `*fmt.wrapError` to `*apperrors.ModelError`).

---

## f) Up to 50 things to get done next

### Critical (wiring gaps)

1. Migrate screenshot.go `AnalyzeStream` error to `classifyModelErr`
2. Migrate screenshot.go `AnalyzeScreenshotImages` error to `classifyModelErr`
3. Migrate screenshot.go `AnalyzeConversation` error to `classifyModelErr`
4. Migrate screenshot.go `AnalyzeConversationStream` error to `classifyModelErr`
5. Delete `wrapWithPrompt` function from vision.go once screenshot.go is migrated
6. Remove `fmt` import from vision.go if no longer needed (check — `String()` method still uses `fmt.Sprintf`)
7. Remove dead `errTestNoop` from model_test.go
8. Replace `wrapNoop` with real `fmt.Errorf("wrapped: %w", err)` chain traversal test

### Consistency fixes

9. Fix `AnalyzeConversationStream` to use `validateAnalyzeInput` instead of inline validation
10. Audit all `fmt.Errorf` error wrapping in screenshot.go (`AnalyzeScreenshot`, `AnalyzeScreenshots`) for classification opportunities
11. Ensure all screenshot.go file-load errors are classified or explicitly left as config errors

### Testing

12. Add BDD (Ginkgo) spec for error classification in agent_bdd_test.go
13. Add BDD spec for stream error classification
14. Add test for `AnalyzeBatch` returning classified per-image errors
15. Add test for `AnalyzeConversation` error classification
16. Add test for `AnalyzeConversationStream` error classification
17. Add test verifying hooks `OnError` receives `*ModelError`
18. Add consumer-pattern example test (extract Kind, decide retry vs. fail)
19. Add test for prompt truncation in `ModelError.Error()` at exact boundary
20. Add test for `KindNetwork` classification (transport error with status 0)
21. Reduce 6s test latency — configure mock to skip fantasy retry
22. Add benchmark for `Classify()` hot path

### Error classification refinement

23. Consider `KindNotImplemented` for HTTP 501
24. Consider `KindServiceUnavailable` for HTTP 503 (currently lumped into `KindServerError`)
25. Add classification for `fantasy.RetryError` — currently only works if RetryError wraps a ProviderError; direct RetryError with non-provider errors falls through to `KindUnknown`
26. Add classification for `io.EOF` / `io.ErrUnexpectedEOF` without ProviderError wrapper
27. Consider `KindContentFilter` for provider content-policy rejections (some providers return 400 with specific messages)

### CLI

28. Add `--retry` flag that uses `IsRetryable()` to auto-retry transient errors
29. Add `--max-retries` flag wired to `fantasy.WithMaxRetries`
30. Add exit code differentiation (e.g. exit 2 for retryable, exit 3 for auth)
31. Add `--verbose` flag showing full error chain including StatusCode
32. Test CLI error output format (currently untested — no CLI tests exist)

### Documentation

33. Update AGENTS.md Sentinel Errors section with ErrorKind system
34. Update AGENTS.md Type Model section with ModelError
35. Add CHANGELOG.md entry for error classification system
36. Add consumer usage example to pkg/vision package doc comment
37. Add `examples/error-handling/main.go` showing `errors.AsType[*vision.ModelError]` pattern
38. Document hooks behavior change (OnError now receives classified error)
39. Update README.md error handling section if one exists

### Batch / Conversation

40. Add error classification test for `AnalyzeBatch` with mixed success/failure
41. Add error classification test for `AnalyzeConversation` with invalid conversation state
42. Consider classified errors in `BatchResult.Err` — should consumers be able to `errors.AsType` them?

### Code quality

43. Run `golangci-lint run --fix` on entire repo to resolve gci import ordering holistically
44. Address pre-existing `err113` warnings in errors_test.go (use static sentinels in test assertions)
45. Address pre-existing `exhaustruct` warnings in cmd/vision/main.go Config construction
46. Run `gofmt -w` on entire repo
47. Add `//go:generate` directive or makefile target for error kind string generation if the enum grows

### Future features

48. Add `RetryPolicy` struct to Config with `MaxAttempts`, `InitialBackoff`, `MaxBackoff`, `RetryableKinds`
49. Add metrics integration via hooks (count errors by Kind)
50. Add structured logging integration that logs Kind, StatusCode, Op on every classified error

---

## g) Questions I cannot figure out myself

### Q1: Should screenshot.go config-validation errors be classified?

The 4 remaining `wrapWithPrompt` calls in screenshot.go wrap errors from `sa.agent()` — which fails on **config validation** (nil model, bad temperature), not model invocation. Should these be:

- (a) Left as plain wrapped errors (config errors aren't model errors), or
- (b) Classified as `KindUnknown` ModelErrors for uniform consumer handling?

### Q2: Is the 6-second test latency from fantasy's retry acceptable?

Tests injecting 429/500 ProviderErrors trigger fantasy's built-in retry middleware (3 retries with backoff), causing 6s delays. Should I:

- (a) Accept it (realistic behavior), or
- (b) Add `fantasy.WithMaxRetries(0)` to test agents to isolate classification logic?

### Q3: Should the CLI auto-retry on transient errors?

The `IsRetryable()` function exists but the CLI just prints and exits. Should I add a `--retry` flag that auto-retries `KindRateLimited`, `KindTimeout`, `KindServerError`, `KindNetwork` with exponential backoff? Or is retry the consumer's responsibility (keep the SDK simple)?

---

## Resolution (2026-07-27)

Re-verified against code (`grep`, `go test`, CHANGELOG, AGENTS.md) four days
after this report.

| Section                     | Claim in report                                                                                           | Resolution                                                                         | Evidence                                                   |
| --------------------------- | --------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| b) screenshot.go wiring     | 4 of 4 sites still use `wrapWithPrompt`                                                                   | **DONE** — `wrapWithPrompt` deleted repo-wide; `grep wrapWithPrompt *.go` → 0 hits | `pkg/vision/screenshot.go`; TODO_LIST execution 2026-07-27 |
| b) `wrapWithPrompt` cleanup | definition still exists                                                                                   | **DONE** — function removed from `vision.go`                                       | `grep -rn wrapWithPrompt --include="*.go"` → empty         |
| c) AGENTS.md update         | no mention of `ModelError`, `ErrorKind`                                                                   | **DONE** — `AGENTS.md` "Classified Model Errors" section + Type Model entry        | `AGENTS.md:37,130-137`                                     |
| c) CHANGELOG entry          | not started                                                                                               | **DONE** — shipped under `[0.2.0] > Added` ("Classified model errors")             | `CHANGELOG.md` `[0.2.0]`                                   |
| f.1–f.5                     | Migrate screenshot.go 4 sites + delete `wrapWithPrompt` + remove dead imports                             | **DONE**                                                                           | as above                                                   |
| f.9                         | Unify `AnalyzeConversationStream` validation → `validateAnalyzeInput`                                     | **DONE**                                                                           | `2026-07-27_11-49` report §a                               |
| f.7                         | Remove dead `errTestNoop`                                                                                 | **OPEN** — still defined, still unused                                             | `pkg/errors/model_test.go:22`                              |
| f.8                         | Replace `wrapNoop` with real `fmt.Errorf("wrapped: %w", err)`                                             | **OPEN** — still a no-op `return err`                                              | `pkg/errors/model_test.go:316-320`                         |
| c) / f.12                   | BDD (Ginkgo) specs for error classification                                                               | **OPEN** — still testify table-driven only                                         | `pkg/vision/error_classification_test.go`                  |
| c) / f.14                   | `AnalyzeBatch` classified-error test                                                                      | **OPEN** — no batch error test exists                                              | `pkg/vision/batch*test*`                                   |
| c) / f.37                   | `examples/error-handling/main.go`                                                                         | **OPEN** — not created                                                             | `examples/`                                                |
| f.23–f.27                   | `KindNotImplemented`, `KindServiceUnavailable`, `KindContentFilter`, `RetryError`/`io.EOF` classification | **OPEN** — none added; 11 kinds unchanged                                          | `pkg/errors/model.go`                                      |
| f.28–f.32                   | CLI `--retry` / `--max-retries` / exit-code differentiation                                               | **OPEN** — CLI prints advice only, no auto-retry                                   | `cmd/vision/main.go`                                       |

**Net:** the wiring and documentation gaps are closed; the testing, example,
and error-kind-refinement work remains open and is tracked in `TODO_LIST.md`.
