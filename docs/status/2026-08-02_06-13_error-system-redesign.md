# Status Report — 2026-08-02 06:13

> **ANNOTATED 2026-08-16 (docs-health):** P0 items 1–4 and most P1 items
> were closed by the `15:26`/`15:49` follow-up sessions; remaining open work
> is tracked in `TODO_LIST.md` / `ROADMAP.md`.

**Session goal:** Design a superb error system based on `erraudit --type-aware` output (229 errors found, 125 violations).

**Starting HEAD:** `6c744e2 build(deps): bump charm.land/fantasy to v0.40.0 and refresh nixpkgs + flake-parts`

**Working tree:** 7 files modified + 1 new file (uncommitted)

---

## Executive Summary

The `erraudit` report surfaced 125 violations. After thorough analysis, the majority were **false positives** — the tool's `context_loss` detector flags _every_ variable in scope as "lost on error path," including variables that are **results of failed operations** (nil/garbage on the error path). Adding those to error messages would be harmful, not helpful. The `generic_return` warnings (17) suggesting per-function concrete error types are anti-idiomatic Go. I fixed the **4 legitimate issues** and documented the rest as intentional design decisions.

---

## a) FULLY DONE

| Item                                                                                                                 | Evidence                                                                                |
| -------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| `Config.Validate()` enriched — all 6 ranged sentinels now include offending value via `%w` wrapping                  | `pkg/vision/vision.go:121-152`; error.Is still matches, error string is self-diagnosing |
| Bare `return err` in `image.go:134` (LoadImageFromReader failure) wrapped with URL context                           | `pkg/vision/image.go:132-135`                                                           |
| Silently swallowed unmarshal error in `structured.go:141` (streaming final object) now returns `KindStructuredParse` | `pkg/vision/structured.go:140-152` — was a real bug                                     |
| `resp.Body.Close()` redundant closure simplified to `defer resp.Body.Close()`                                        | `pkg/vision/image.go:121`                                                               |
| `cost.go:102` bare `NewAgent` failure wrapped with operation context                                                 | `pkg/vision/cost.go:101-103`                                                            |
| `screenshot.go:149` bare `NewAgent` failure wrapped with "build agent" context                                       | `pkg/vision/screenshot.go:147-150`                                                      |
| 3 BDD tests fixed: `gomega.Equal` → `gomega.MatchError` + `ContainSubstring`                                         | `pkg/vision/agent_bdd_test.go:24-60`                                                    |
| New test file: `TestValidationErrorsIncludeOffendingValues` (7 table cases) + `TestNoModelReturnsBareSentinel`       | `pkg/vision/validation_errors_test.go`                                                  |
| `AGENTS.md` updated with 7 new error-system design decisions + sentinel error docs annotated                         | `AGENTS.md` Key Design Decisions + Sentinel Errors sections                             |
| `go build ./...`                                                                                                     | ✓ exit 0                                                                                |
| `go vet ./...`                                                                                                       | ✓ exit 0                                                                                |
| `gofmt -l .`                                                                                                         | ✓ clean                                                                                 |
| `go test -race ./...`                                                                                                | ✓ all packages pass                                                                     |

---

## b) PARTIALLY DONE

1. ~~**Error system analysis — real issues fixed, false positives documented but not suppressed via linting config.** I documented in AGENTS.md WHY the `context_loss` and `generic_return` findings are false positives, but I did not configure `erraudit` (or any linter) to suppress them. The 108 `context_loss` ERROR-severity findings will reappear on every future run, creating noise.~~ still open — no `.erraudit.yaml`; advisory-vs-gate question (g1) unanswered

2. ~~**`internal/cli/helpers.go:75` context loss — addressed via documentation only.**~~ done at `8ac8dde` — `temperature=%.2f` added to the error message

3. ~~**`cmd/vision/main.go` context losses — not touched at all.**~~ closed as intentional — AGENTS.md decision: result variables never go in error messages; flag errors are self-describing

---

## c) NOT STARTED

1. **No linting integration for the error system.** No `erraudit` config, `.erraudit.yaml`, CI gate, or `//nolint` directives were added. ← still open
2. ~~**No errors.Is/errors.AsType migration audit.** The `hierarchical-errors` skill was loaded but the project already uses `errors.AsType[E]` correctly — no `errors.As` to migrate. This was verified by reading the code but no formal audit tool was run.~~ closed — verified; AGENTS.md documents the `errors.AsType` convention
3. ~~**No nix flake verification** (`nix build`, `nix run .#test`, `nix run .#lint`, `nix flake check`) — only Go toolchain checks were run.~~ done in the `15:49` session — all green
4. ~~**No jsonv2 test verification** (`GOEXPERIMENT=jsonv2 go test ./...`).~~ done in the `15:26` session
5. ~~**No CHANGELOG entry** for the error system improvements.~~ done at `389e788`
6. ~~**No example update** — `examples/error-handling/main.go` was not reviewed for alignment with the new enriched error messages.~~ done at `e1c633d`

---

## d) TOTALLY FUCKED UP

1. **Nothing destructive.** No files damaged, no history rewritten, no broken state.

2. **Minor: `gofmt` violation on first write.** The new test file `validation_errors_test.go` had a struct field alignment issue (`wantSubstr string` vs `wantSubstr  string`). Caught by `gofmt -l .` in the final verification pass and fixed immediately. Should have formatted before committing to the pattern.

3. **Honest gap: I did not verify the streaming unmarshal fix with a test.** The `structured.go` fix (swallowed error → hard error) is the most behaviorally significant change in this session, but I added **zero tests** for it. The existing structured test (`structured_test.go`) only tests the happy path and validation errors — no test mocks a malformed final object from the stream. If someone reverts the fix, no test will catch it.

---

## e) WHAT WE SHOULD IMPROVE

### Error System Specific

1. **Add a test for the structured streaming unmarshal failure path.** This is the highest-risk gap — a real bug fix with zero test coverage. A mock that returns a final `ObjectStreamPartTypeFinish` with a malformed object would prove the error surfaces correctly.

2. **Consider an `erraudit` config file** (`.erraudit.yaml` or similar) to suppress the false-positive categories:
   - `context_loss` on result variables (`decoded`, `data`, `img`, `jsonBytes`, `encoded`, `resized`)
   - `generic_return` on public API functions (idiomatic Go returns `error`, not concrete types)

3. **The `cmd/vision/main.go:120` flag-parse error could include the flag name.** Currently `fmt.Errorf("parse flags: %w", err)` — the `flag.FlagSet` error itself usually names the flag, so this is adequate but could be more structured.

4. **Consider a centralized `wrapSentinel(sentinel, value, want)` helper** to standardize the `"sentinel: got %v, want %v"` pattern. Currently each call site inlines `fmt.Errorf("%w: got %.2f, want [0.0, 2.0]", ...)`. A helper would prevent format drift.

5. **The `apperrors.Wrap` function is only used in structured.go.** Other model-error sites use `classifyModelErr`. Consider whether `Wrap` should be used more broadly or whether the two helpers should be consolidated.

### Process-Level

6. **Always format new files before declaring done.** The `gofmt` miss was avoidable.

7. **Always add a test for behavior-changing fixes** — especially when converting a silent swallow to a hard error.

8. **Run the full nix verification matrix** when touching error paths that cross package boundaries.

---

## f) Next Actions (up to 50)

### P0 — Close critical gaps from this session

1. ~~**Write a test for `AnalyzeStructuredStream` final-object unmarshal failure.**~~ done at `7cc90bc` — `TestAnalyzeStructuredStreamUnmarshalFailure`
2. ~~**Run `GOEXPERIMENT=jsonv2 go test ./...`**~~ done in the `15:26` session; permanent CI job since
3. ~~**Run `nix run .#test`**~~ done in the `15:26` session
4. ~~**Run `nix run .#lint`**~~ done in the `15:26` session (0 issues)

### P1 — Error system hardening

5. Add an `erraudit` config or `//nolint` directives to suppress false-positive `context_loss` and `generic_return` findings, so the tool output becomes actionable. ← still open
6. ~~Extract a `wrapSentinel(sentinel error, got, want string) error` helper to standardize the validation error format and prevent drift.~~ **Won't implement — evaluated and rejected in the `15:26` session** (per-site format verbs differ; 6 inline calls are table-tested)
7. ~~Add tests for `LoadImageFromURLWithClient` that verify the URL appears in all error paths~~ done at `8ac8dde` (3 subtests)
8. ~~Review `examples/error-handling/main.go` and update it~~ done at `e1c633d`
9. ~~Add a CHANGELOG entry for the error system improvements.~~ done at `389e788`
10. ~~Audit `internal/cli/helpers.go:75` — consider including `temperature`~~ done at `8ac8dde`
11. ~~Consider adding structured fields to `ModelError` for `Op`, `Prompt`, and `StatusCode`~~ already present — the fields exist on `ModelError`; only a godoc example is missing (see #22)
12. ~~Add `errors.Is` tests for the enriched sentinels at the `pkg/errors` level~~ done at `8ac8dde` (12 table cases)

### P2 — Broader error system improvements

13. **Consider a `ValidationError` type** that carries `Field string`, `Value any`, `Constraint string` — consumers could render form-validation UI from it. Currently the value is baked into the string via `fmt.Errorf`.
14. **Consider `RetryAdvice` on `ModelError`** — a structured hint (`"retry after Xs with backoff"`) derived from `RetryAfter` + `Kind`, so consumers don't have to implement the logic themselves.
15. **Consider HTTP error mapping** — a `func (e *ModelError) HTTPStatus() int` that maps `ErrorKind` → HTTP status for HTTP API consumers.
16. **Add error wrapping to `WithRetry`** — currently returns raw `ctx.Err()` with `//nolint:wrapcheck`. Consider wrapping with attempt count for debugging.
17. **Add `Unwrap() []error` to `ModelError`** if it ever needs to carry multiple causes (e.g., batch analysis).
18. **Review all `//nolint:wrapcheck` directives** — are they still needed after the error system improvements?
19. **Add error sentinel for retry exhaustion** — `ErrRetriesExhausted` wrapping `lastErr`, so consumers can distinguish "failed after N retries" from "failed immediately".
20. **Consider `apperrors.Join(errs ...error) error`** for batch analysis that collects per-image errors.
21. ~~**Document the error taxonomy in a diagram** — sentinels vs ModelError vs wrapped errors, when each appears.~~ done at `5bc97b4` — `docs/ERROR_DESIGN.md`
22. **Add a `godoc` example** for `pkg/errors` showing `errors.AsType[*ModelError]` extraction + `IsRetryable()` check.
23. **Add a `godoc` example** for `pkg/vision` showing `errors.Is(err, vision.ErrInvalidTemperature)` with enriched message.

### P3 — Verification and CI

24. ~~Run `go test -race -coverprofile=coverage.out ./...` and check error-path coverage.~~ done in the `15:26`/`15:49` sessions (coverage 90.1% in `pkg/vision`)
25. Add `erraudit` as a flake app (`nix run .#erraudit`) if it's useful, or document why it's excluded. ← still open
26. ~~Add a CI gate on `golangci-lint run ./...` with the project's specific linter config.~~ exists — `lint` job in `.github/workflows/ci.yml`
27. Review whether `erraudit --type-aware` should be a CI gate or an advisory tool. ← still open (needs user decision)
28. ~~Run `nix flake check` to confirm the flake still builds after uncommitted changes.~~ done in the `15:49` session
29. ~~Commit the error system work with a descriptive message.~~ done (`23d74ec`…`5bc97b4`)
30. ~~Review `go.mod` — no dependency changes needed for this work, but confirm.~~ confirmed — no changes needed

### P4 — Documentation and examples

31. Update `README.md` error-handling section with the new enriched messages. ← still open (README documents classified errors but not the enriched format; no `docs/ERROR_DESIGN.md` link)
32. ~~Update `docs/DOMAIN_LANGUAGE.md` if it references error terminology.~~ done — `ErrorKind` (16 kinds), `ModelError`, `Classify` defined
33. ~~Add a `docs/ERROR_DESIGN.md` documenting the full error taxonomy and design decisions.~~ done at `5bc97b4`
34. ~~Update `FEATURES.md` if error classification is listed as a feature.~~ done — FEATURES.md "Error Handling" section
35. ~~Create a comprehensive error-handling example in `examples/error-handling/main.go`.~~ done at `e1c633d`
36. ~~Document the `erraudit` false-positive categories in AGENTS.md "Gotchas".~~ done — AGENTS.md error-handling decisions
37. ~~Add the validation error format (`"sentinel: got %v, want ..."`) to AGENTS.md code conventions.~~ done — "Validation errors include offending values"
38. Review `examples/structured-stream/main.go` for alignment with the new unmarshal error behavior. ← still open

### P5 — Deep error system audit (future)

39. Evaluate `cockroachdb/errors` (banned list says use instead of `pkg/errors`) — does the project need it? Currently uses stdlib only.
40. Evaluate whether `ModelError` should implement `GRPCStatus()` for gRPC consumers.
41. Consider a `fmt.Stringer` implementation for `ErrorKind` that returns a human-readable description (e.g., `"Rate Limited — the provider returned 429 Too Many Requests"`).
42. Consider `ErrorKind.MarshalJSON()` for structured logging (currently it's a string, so it serializes, but a typed enum would be cleaner).
43. Audit all `errors.New(...)` calls in the codebase — are any using string matching instead of sentinels?
44. Review the `isContentFilterRejection` heuristic — could it be a typed error from fantasy instead of string scanning?
45. Consider adding `ModelError.WithCause(err)` builder for chained wrapping.
46. Review whether `CostTracker` errors need sentinel classification (currently returns raw errors).
47. Evaluate error propagation in `Conversation` methods — any silent swallows?
48. Review `validate.go` — `ValidateImage` returns `ErrInvalidImage` but doesn't include the detected format or byte prefix.
49. Consider `ErrUnsupportedImageFormat` as a distinct sentinel from `ErrInvalidImage`.
50. Run a full `brutal-self-review` or `full-code-review` skill pass focused on error handling.

---

## g) Questions I CANNOT Answer Myself

1. **Should `context_loss` findings on `cmd/vision/main.go` (the CLI) be fixed or suppressed?** The CLI is a thin layer over the SDK. Including 11 flag values in a flag-parse error message would be noise, but the `erraudit` tool flags them at ERROR severity. I need to know: does the project treat `erraudit` as a CI gate (must be zero) or an advisory tool? If it's a gate, I need a suppression strategy.

2. **Should the structured streaming partial-object unmarshal failures (during `ObjectStreamPartTypeObject`, not just `Finish`) also be hard errors?** Currently they are best-effort (silently skipped). The final object unmarshal was changed to a hard error. The partial objects during streaming are arguably less critical — a transient parse failure on a partial shouldn't kill the stream. But I'm not sure if the domain requires all-or-nothing semantics for structured streaming.

3. **Is `erraudit` a tool the user actively uses for CI gating, or just for one-off analysis?** This determines whether I should invest in `.erraudit.yaml` suppression config or just document the false positives and move on. If it's not in CI, the 125 violations are informational and the 4 real fixes are sufficient.

---

## Verification Snapshot

| Check       | Command                             | Result      |
| ----------- | ----------------------------------- | ----------- |
| Build       | `go build ./...`                    | ✓           |
| Vet         | `go vet ./...`                      | ✓           |
| Format      | `gofmt -l .`                        | ✓ clean     |
| Test (race) | `go test -race ./...`               | ✓ all pass  |
| Nix build   | `nix build .`                       | **not run** |
| Nix test    | `nix run .#test`                    | **not run** |
| Nix lint    | `nix run .#lint`                    | **not run** |
| Flake check | `nix flake check`                   | **not run** |
| jsonv2 test | `GOEXPERIMENT=jsonv2 go test ./...` | **not run** |

**Notable gap:** The most behaviorally significant change (streaming unmarshal error) has **no dedicated test**. This is the highest-priority follow-up.

---

## Files Changed This Session

```
 M AGENTS.md                              (+15 design decisions, sentinel docs)
 M pkg/vision/agent_bdd_test.go           (3 tests: Equal → MatchError + Contains)
 M pkg/vision/cost.go                     (+1 import, wrapped bare error)
 M pkg/vision/image.go                    (fixed bare propagation, simplified Close)
 M pkg/vision/screenshot.go               (wrapped bare agent() error)
 M pkg/vision/structured.go               (swallowed error → hard error)
 M pkg/vision/vision.go                   (Validate() enriched with offending values)
?? pkg/vision/validation_errors_test.go   (new: 8 table-driven test cases)
```

---

_Generated 2026-08-02 06:13. Based solely on this session's work and observations._
