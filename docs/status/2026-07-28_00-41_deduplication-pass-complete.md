# Status Report: Deduplication Pass — 2026-07-28

**Session:** Deduplication run (`art-dupl --type-aware --sort total-tokens -t 1 --html`)
**Result:** 10 clone groups → 5 (all remaining accepted with rationale)
**Tests:** All 81 pass with `-race`

---

## a) FULLY DONE

### Refactors shipped (production code)

| Change                                           | File                           | Effect                                                                                                                                                                                                            |
| ------------------------------------------------ | ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Extracted `invalidate()` helper                  | `pkg/vision/screenshot.go:124` | Removed 11x `sa.cachedAgent = nil; return sa` clone                                                                                                                                                               |
| Extracted `prepare()` + `preparedRequest` struct | `pkg/vision/vision.go`         | Collapsed 6x duplicated prologue (validate + preprocess + fireStart + timeout + toFileParts) across Analyze/AnalyzeStream/AnalyzeConversation/AnalyzeConversationStream/AnalyzeStructured/AnalyzeStructuredStream |
| Extracted `finishResult()` helper                | `pkg/vision/vision.go`         | Collapsed 4x duplicated epilogue (build AnalyzeResult + fireFinish + return)                                                                                                                                      |
| Extracted `buildObjectCall[T]()` generic helper  | `pkg/vision/structured.go`     | Removed 2x identical 18-line ObjectCall construction                                                                                                                                                              |
| Aligned all `ctx` → `prep.ctx` in structured.go  | `pkg/vision/structured.go`     | Ensured timeout-applied context used in all hook fires and model calls                                                                                                                                            |

### Documentation

- Created `dedup-acceptance.md` at repo root documenting every accepted clone group with one-line rationale.

### Verification

- `go build ./...` — clean
- `go test -race ./... -count=1` — 81/81 pass
- `go vet ./...` — clean
- `gofmt -l .` — clean (no diff)
- `art-dupl` re-run confirms 5 accepted groups remain

---

## b) PARTIALLY DONE

Nothing. Every refactor I started is complete and tested.

---

## c) NOT STARTED

Nothing within the deduplication scope. All 10 original clone groups were either eliminated or explicitly accepted.

---

## d) TOTALLY FUCKED UP

### Bug I introduced and caught immediately

- **`invalidate()` infinite recursion** — When using `edit` with `replace_all: true` to replace all 11 occurrences of `sa.cachedAgent = nil / return sa`, the replacement also matched the body of the newly-added `invalidate()` method itself, creating `func invalidate() { return sa.invalidate() }`. Caught by `go test` (stack overflow), fixed before proceeding. Lesson: `replace_all` is dangerous right after adding a helper that contains the same pattern. Should have added the helper AFTER the bulk replace, or used a more specific `old_string`.

### Pre-existing test modification (not mine)

- `pkg/vision/retry_test.go` had unstaged changes at session start (MaxRetries: 1 → 0 across 4 tests, aligning with the MaxRetries=0 default documented in AGENTS.md). I left these untouched per the safety rule "never revert changes you didn't author." One initially showed FAIL due to a stale build cache; passed on re-run.

---

## e) WHAT WE SHOULD IMPROVE

### Code quality observations (session-accumulated)

1. **`prepare()` returns a struct with `cancel` exposed** — Callers must remember `defer prep.cancel()`. A `func()` return or a closer pattern would be slightly safer but less readable. Current form is the Go-idiomatic tradeoff.

2. **`finishResult()` doesn't fire `OnError`** — Classification + `fireError` is still inlined in each method because the error label/message differs per call site ("vision agent generate" vs "vision agent conversation stream"). This is correct but leaves 6 similar error-handling blocks. Could be unified with a label parameter but the dedup tool doesn't flag them.

3. **`buildObjectCall[T]()` uses `reflect.TypeOf(*new(T))`** — Idiomatic Go trick to get a type without declaring a variable. Works but `var zero T; reflect.TypeOf(zero)` is more readable for newcomers. Tradeoff; kept compact form.

4. **Structured methods still have 2 clone groups** — The `if err != nil { return nil, err }` + `defer prep.cancel()` pattern. This is irreducible Go boilerplate, not harmful duplication.

5. **Examples have 3 clone groups** — CLI setup boilerplate + UIReview struct. Accepted because examples must be self-contained teaching programs.

### Process improvements

6. **Should have verified the pre-existing `retry_test.go` changes more carefully** before running tests — I was briefly confused by the FAIL output. A `git status` at session start would have surfaced it immediately.

7. **Should have added `invalidate()` AFTER the bulk replace**, not before, to avoid the recursion bug.

---

## f) Up to 50 things to get done next

Ranked by impact:

### High impact — code quality

1. **Run `golangci-lint run ./...`** — Matches the flake `lint` app; may surface issues `go vet` misses
2. **Review `optionalParams()` pointer aliasing** — Returns pointers into `Config` fields; if Config is mutated after Agent construction, the pointers could dangle. Verify lifecycle.
3. **Add integration test for `prepare()` + `finishResult()` path** — Currently covered transitively via BDD; a focused unit test would catch regressions faster
4. **Consider `preparedRequest` as a `Closer`** — Implement `io.Closer` so callers can `defer prep.Close()` instead of `defer prep.cancel()`
5. **Unify error classification** — Extract `classifyAndFire(ctx, label, prompt, err)` to collapse the remaining 6 error blocks (out of dedup scope but would improve consistency)

### Medium impact — documentation

6. **Update `AGENTS.md` Key Design Decisions** — Add entries for `prepare()`, `finishResult()`, `invalidate()`, `buildObjectCall[T]()`
7. **Update `docs/DUPLICATION_POLICY.md`** — Reference the new helpers as examples of how duplications were resolved
8. **Add architecture diagram** — The `prepare()` → `buildAgentCall` → `generate` → `finishResult()` flow is now the spine of every analysis method; a diagram would help onboarding
9. **Document the `preparedRequest` contract** — The cancel-on-defer requirement is critical; a doc comment on the type would help

### Medium impact — testing

10. **Add a test for cache invalidation** — Verify that calling any `With*` method actually clears `cachedAgent` (currently only tested implicitly via ScreenshotAnalyzer tests)
11. **Add a test for `buildObjectCall[T]()` directly** — Verify schema generation and param forwarding without going through the full AnalyzeStructured path
12. **Add a test for `finishResult()` hook firing** — Verify OnFinish receives the correct Text/Usage/RawResponse
13. **Add benchmarks for the new helpers** — Ensure no allocation regressions vs. the inline code

### Lower impact — polish

14. **Consider naming `prep` → `prepared`** — More descriptive; `prep` is terse but slightly opaque
15. **Review all doc comments for the new helpers** — Ensure they explain WHY, not just WHAT
16. **Consider extracting `streamDeltaHandler`** — The `OnTextDelta` closure construction is similar in AnalyzeStream and AnalyzeConversationStream
17. **Review `optionalModelParams` struct** — Could use generics or a builder pattern for type safety
18. **Add `//nolint` comments where needed** — Ensure linter compliance for the new helpers
19. **Review examples for consistency** — Some use `cli.NewAgent`, others use `vision.NewAgent` directly; document when to use which
20. **Consider a `examples/shared` package** — If example duplication bothers future reviewers, a shared setup package could reduce it (but breaks the "self-contained" property)

### Exploration

21. **Investigate whether `fantasy` has a built-in `Closer` or `Finalizer` pattern** — Could simplify the `prepare/cancel` dance
22. **Check if `fantasy.AgentResult` could expose a `ToAnalyzeResult()` helper** — Would eliminate the need for `finishResult()` entirely
23. **Review the `WithErrorContext` pattern** — Could the error label be auto-derived from the call stack?

### Housekeeping

24. **Commit the changes** — All refactors are uncommitted; user hasn't requested commit
25. **Review `dedup-acceptance.md` location** — Root vs. `docs/`; check if other docs reference it
26. **Verify `docs/status/` directory is gitignored or tracked** — Confirm convention
27. **Clean up any orphaned imports** — Verify no unused imports after refactors (go build would catch this, but double-check)
28. **Review the `internal/visionutil` package** — `AppendSystemAndPrompt` and `UnmarshalToType` are used by structured.go; ensure they're still needed
29. **Check if `schema.Generate` could be cached** — Called on every `buildObjectCall[T]()` invocation; if T is the same across calls, caching could help
30. **Review `classifyModelErr` signature** — Takes a string label; could be an enum for type safety

### Future-proofing

31. **Consider a `PrepareOption` functional-options pattern** — If `prepare()` needs to grow (e.g., per-call hooks), options would keep it extensible
32. **Document the `Analyzer` interface contract** — The `prepare()` extraction assumes all implementations follow the same prologue; mock implementations need to know this
33. **Review thread safety of `preparedRequest`** — It's used within a single goroutine per analysis call, but document this explicitly
34. **Consider a `Result` interface** — If `AnalyzeResult` and `ObjectResult[T]` need shared behavior, an interface could unify them
35. **Review error wrapping depth** — The classify → wrap → fire chain adds layers; ensure `errors.Is` and `errors.AsType` still work through the stack
36. **Add a `CHANGELOG.md` entry** — If the project maintains one, document the internal refactors
37. **Review the `Retry` composition** — `generate()` and `generateObject()` have parallel retry logic; could be unified with a generic helper
38. **Check if `WithRetry` could be a decorator** — Instead of conditional logic in `generate()`, wrap the agent in a retry decorator
39. **Review the `Hooks` firing order** — Ensure `fireStart` → `fireError`/`fireFinish` is always correct across all paths
40. **Consider structured logging for the new helpers** — `prepare()` and `finishResult()` are good candidates for debug-level logging
41. **Review the `CostTracker` integration** — It hooks into `OnFinish`; verify it still works with the new `finishResult()` path
42. **Add property-based tests for `prepare()`** — Verify it handles edge cases (nil images, empty prompt, nil context)
43. **Review the `PreprocessConfig` application** — It's called in `prepare()`; verify the resize logic is correct
44. **Check if `toFileParts` could be lazy** — Currently eagerly converts all images; for large batches, lazy conversion could save memory
45. **Review the `MediaType` type** — It's a string type; consider if it should be an enum or have validation
46. **Consider a `Builder` pattern for `Config`** — The `ScreenshotAnalyzer` has fluent setters; `Config` could benefit from the same
47. **Review the `Conversation` type** — It's used by `prepare()` indirectly; ensure it's thread-safe
48. **Check if `AnalyzeBatch` could use `prepare()`** — It calls `Analyze` per image, which already uses `prepare()`; verify no double-validation
49. **Review the `examples/openrouter` example** — It uses `ScreenshotAnalyzer` directly while others use `Agent`; document the difference
50. **Run the nix flake checks** — `nix run .#test` and `nix run .#lint` to verify reproducibility

---

## g) Questions (cannot figure out myself)

### Q1: Should I commit the refactors?

All changes are uncommitted. The AGENTS.md says "An auto-git commit daemon runs continuously and commits changes automatically" — but nothing has been committed this session. Should I:

- **(a)** Commit now with a descriptive message?
- **(b)** Leave it for the auto-git daemon?
- **(c)** Wait for explicit user instruction?

### Q2: Is the `retry_test.go` modification intentional?

The file has unstaged changes (MaxRetries: 1 → 0 across 4 tests) that align with the AGENTS.md note about MaxRetries=0 default. I left them untouched per safety rules. Were these changes yours, or should I investigate further?

### Q3: Should the accepted duplications in `dedup-acceptance.md` be excluded via `--exclude-pattern` instead?

The remaining 5 clone groups will reappear on every future `art-dupl` run. I documented them as accepted, but the report will still show them. Should I:

- **(a)** Leave as-is (documented acceptance, report shows them)?
- **(b)** Add `--exclude-pattern` flags to suppress them in future runs?
- **(c)** Something else?

---

## Summary

**The deduplication goal is achieved: zero harmful duplication remains.** Every eliminated clone was a real maintenance burden (scattered cache invalidation, copy-pasted prologues/epilogues, duplicated ObjectCall construction). Every accepted clone is either irreducible Go idiom or intentionally self-contained example code. All 81 tests pass with the race detector enabled.
