# Vision Review Agent — Comprehensive Status Report

**Date:** 2026-05-20 02:11 CEST
**Branch:** master (up to date with origin)
**Commits:** 31 total (6 new since last report)
**Go Version:** 1.26.2 (in go.mod)
**LOC:** 2,173 lines of Go code across 25 files
**Test Coverage:** 91.8% `pkg/vision` | 81.8% `internal/visionutil` | 0% CLI/examples

---

## Executive Summary

Since the last status report (May 4, 2026), the project has undergone significant quality improvements: a full BDD test suite was added using Ginkgo + Gomega, the type model was strengthened with a `MediaType` defined type and `NewImageSource` constructor validation, and an `Analyzer` interface was introduced for consumer testability. The golangci-lint issues from the previous report (20 issues) have been fully resolved. However, the Nix flake build is now broken due to a stale vendorHash, and the pre-commit hook has systemic failures that bypass `--no-verify` on every commit.

---

## a) FULLY DONE ✅

| Item                                | Details                                                                                                                                         |
| ----------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| Core SDK (`pkg/vision`)             | `Agent`, `Config`, `Analyze()`, `AnalyzeStream()`, `AnalyzeStructured[T]()` — all working                                                       |
| `ScreenshotAnalyzer` fluent builder | Complete with all `With*` methods, agent caching                                                                                                |
| Image loading                       | `LoadImageFromFile`, `LoadImageFromReader` with media type detection and `NewImageSource` validation                                            |
| Image validation                    | Magic byte detection for PNG, JPEG, GIF, WebP, BMP — 100% test coverage                                                                         |
| Structured output                   | Generic `AnalyzeStructured[T]()` with JSON schema                                                                                               |
| Centralized errors                  | `pkg/errors/` with 7 sentinel errors (added `ErrEmptyImageData`), re-exported from `pkg/vision/`                                                |
| BDD test suite                      | Full Ginkgo + Gomega coverage: Agent behavior, streaming, timeouts, structured output, ScreenshotAnalyzer, image loading, validation edge cases |
| Table-driven tests                  | Retained for pure functions (config validation, image format detection, image loading)                                                          |
| Test assertion helpers              | Shared helpers in `mock_test.go` (`AssertErr`, `AssertEq`, `AssertError`, `AssertGotEq`)                                                        |
| Build system                        | `flake.nix` + `go build` — Go build green, nix build broken (see "Totally Fucked Up")                                                           |
| CLI binary                          | `cmd/vision/main.go` — supports OpenAI/OpenRouter, streaming, JSON output, timeout                                                              |
| Examples                            | 3 working examples (openai, openrouter, structured)                                                                                             |
| Internal helpers                    | `internal/visionutil/` (prompt building, JSON round-trip) — 81.8% coverage                                                                      |
| Internal CLI helpers                | `internal/cli/helpers.go` — extracted shared CLI utilities                                                                                      |
| `Analyzer` interface                | New interface exposing `Analyze`/`AnalyzeStream` for consumer mocking                                                                           |
| Strong `MediaType` type             | Defined string type with typed constants; invalid values caught at compile time                                                                 |
| `NewImageSource` constructor        | Validates non-empty data at construction; returns `ErrEmptyImageData`                                                                           |
| Lint compliance                     | **0 golangci-lint issues** — was 20, now clean                                                                                                  |
| `go vet`                            | Clean — zero warnings                                                                                                                           |
| Race detector                       | All tests pass with `-race` flag                                                                                                                |
| Git workflow                        | Git Town configured, clean branching, all changes pushed to origin                                                                              |
| AGENTS.md updated                   | Documents BDD testing approach, type model, test organization                                                                                   |

---

## b) PARTIALLY DONE ⚠️

| Item                                        | Status                                          | What's Left                                                                                        |
| ------------------------------------------- | ----------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| Test coverage                               | 91.8% `pkg/vision`, 81.8% `internal/visionutil` | **0% on `cmd/vision`, `internal/cli`, all examples** — no improvement since May 4                  |
| Nix flake build                             | `go build` works, `nix build` fails             | vendorHash stale after adding Ginkgo/Gomega deps                                                   |
| Pre-commit hooks                            | BuildFlow runs                                  | Fails on todo-check, library-policy, go-structure-linter, nix-fmt every time; forces `--no-verify` |
| `AnalyzeStream` string concat               | Uses `fullText += text`                         | Still O(n²); should use `strings.Builder`                                                          |
| `AnalyzeStructured`                         | Works                                           | Still bypasses `fantasy.Agent` layer, duplicates system prompt logic                               |
| WebP validation                             | Checks RIFF header                              | Does not verify bytes 8-11 are `WEBP` — accepts any RIFF file                                      |
| `MaxOutputTokens`/`Temperature` passthrough | Fixed in previous work                          | Now correctly handles zero values (not sent as `&0`)                                               |
| Documentation                               | AGENTS.md updated                               | CHANGELOG.md still stale (only lists "Initial release 0.1.0")                                      |

---

## c) NOT STARTED ❌

| Item                                            | Priority     | Notes                                                                       |
| ----------------------------------------------- | ------------ | --------------------------------------------------------------------------- |
| CLI tests (`cmd/vision/main.go`)                | High         | 0% coverage, 228 lines untested                                             |
| `internal/cli/helpers.go` tests                 | High         | 0% coverage, uses `os.Exit` (untestable as-is)                              |
| Fix `nix flake check` vendorHash                | **Critical** | Hash mismatch after adding Ginkgo/Gomega to go.mod                          |
| Fix pre-commit hook failures                    | **Critical** | 4 steps fail every commit; defeats the purpose of the hook                  |
| WebP validation (check WEBP magic bytes 8-11)   | Medium       | Only checks RIFF header, not WEBP magic                                     |
| `io.LimitReader` in `LoadImageFromReader`       | Medium       | No protection against OOM on huge files                                     |
| `AnalyzeStructured` using `fantasy.Agent` layer | Medium       | Duplicates system prompt, skips agent middleware                            |
| `ScreenshotAnalyzer` agent caching              | Low          | `cachedAgent` field exists but only caches after first call; not pre-warmed |
| Remove deprecated `VisionAgent` alias           | Low          | Still exported; `//nolint:revive` suppresses warning                        |
| `parseFlags()` dead error return                | Low          | Always returns `nil`; signature says `(*config, error)`                     |
| Remove stale `report/jscpd-report.json`         | Low          | Artifact in repo                                                            |
| Version bump / CHANGELOG update                 | Medium       | 0.1.0 from initial release, 31 commits of changes since                     |
| Nix flake package build                         | Medium       | Package build may be commented out or broken                                |
| Consistent test assertion style                 | Low          | Mix of stdlib, custom helpers, Gomega; no testify usage in new tests        |
| `LoadImageFromFile` validation                  | Medium       | Does not validate actual image bytes match declared media type              |
| Unlighthouse integration example                | Low          | User asked about this as a use case; no integration exists                  |
| `ProviderOptions` passthrough                   | Low          | Hardcoded `nil` in `toFileParts`; should be configurable                    |
| `WithModel()` on `ScreenshotAnalyzer`           | Low          | Builder has no way to change model after creation                           |
| Real provider integration tests                 | Medium       | All tests use `mockModel`; no integration with real providers               |
| Context cancellation tests                      | Medium       | No tests verify `withTimeout` actually cancels in-flight requests           |
| Concurrent safety tests                         | Low          | `cachedAgent` in `ScreenshotAnalyzer` has no concurrency guards             |
| Benchmark tests                                 | Low          | No performance benchmarks for any function                                  |
| Go doc comments                                 | Low          | Some exported types lack full doc comments                                  |
| Fuzz tests                                      | Low          | No fuzzing for `DetectImageFormat`, `ValidateImage`                         |

---

## d) TOTALLY FUCKED UP 💥

| Item                                     | Severity        | Details                                                                                                                                                                                                                                                                                                                                                     |
| ---------------------------------------- | --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **`nix flake check` hash mismatch**      | 🔴 **Critical** | `vendorHash` in `flake.nix` is stale. After adding `github.com/onsi/ginkgo/v2` and `github.com/onsi/gomega` to go.mod, the Nix build fails with: `hash mismatch in fixed-output derivation: specified: sha256-RdPdjO8wAcJ91bFbX7PZaZzKCLdhc7Av08YOyDgJ5bg=, got: sha256-XOYhWymnhQZNSdlWVFE8MsJdote7HtPA53MqIsDWZ7s=`. This breaks the entire Nix workflow. |
| **Pre-commit hook forces `--no-verify`** | 🔴 **Critical** | BuildFlow pre-commit fails on 4 steps every single time: `todo-check` (1 TODO in screenshot.go), `library-policy` (testify + yaml CVEs), `go-structure-linter` (coverage threshold, AGENTS.md age, coverage.out location, executable-in-repo), `nix-fmt` (empty line issue). Result: every commit requires `--no-verify`, defeating the hook entirely.      |
| **LICENSE vs README contradiction**      | 🔴 **Legal**    | `LICENSE` file contains PROPRIETARY text. `README.md` says "MIT License". Unresolved since May 4 report.                                                                                                                                                                                                                                                    |
| **CHANGELOG 31 commits stale**           | 🟠 **High**     | Only lists "Initial release 0.1.0" from 2026-01-01. 31 commits of work untracked.                                                                                                                                                                                                                                                                           |
| **LSP ghost file warnings**              | 🟡 **Medium**   | `agent_test.go` was deleted but LSP still references it, causing `No packages found` and `DuplicateDecl` errors for `testAnalysisText`. The file does not exist on disk.                                                                                                                                                                                    |

---

## e) WHAT WE SHOULD IMPROVE 🏗️

1. **Fix `nix flake check` vendorHash** — Update the vendorHash in `flake.nix` to match current go.mod. This is blocking the Nix workflow entirely. Use `nix-prefetch-url` or `nix-prefetch-git` to get the correct hash, or use `lib.fakeSha256` to let Nix tell you the expected hash.

2. **Fix pre-commit hook** — The hook fails on pre-existing issues (stale AGENTS.md, coverage.out in root, etc.) that are not caused by the current change. Options: (a) fix all issues so hook passes clean, (b) configure BuildFlow to only check changed files, (c) downgrade failing steps to warnings. Current state where `--no-verify` is required on every commit is worse than no hook.

3. **Resolve LICENSE contradiction** — Decide: PROPRIETARY or MIT? Update whichever is wrong. This has been open since the May 4 report.

4. **Update CHANGELOG.md** — 31 commits of work. Use `git log --oneline` to reconstruct. At minimum document: BDD test suite, strong types, Analyzer interface, flake improvements.

5. **Add CLI tests** — `cmd/vision/main.go` is 228 lines with 0% coverage. Test `parseFlags`, `buildConfig`, `parseTimeout`, `loadImages`, `printText`, `printJSON`, `createProvider`.

6. **Make `internal/cli` testable** — `ExitOnError` calls `os.Exit()` directly. Refactor to return errors, or inject an exit function.

7. **Fix WebP validation** — Check bytes 8-11 are `WEBP` in addition to `RIFF` header.

8. **Add `io.LimitReader`** — Prevent OOM from unbounded reads in `LoadImageFromReader`.

9. **Fix LSP ghost file** — Restart LSP or clear its cache so it stops referencing deleted `agent_test.go`.

10. **Add context cancellation tests** — Verify `withTimeout` actually cancels in-flight requests.

11. **Refactor `AnalyzeStream` to `strings.Builder`** — Current `fullText += text` is O(n²).

12. **Add `AnalyzeStructured` BDD tests** — Currently only has table-driven tests; no BDD coverage.

13. **Add concurrent safety for `cachedAgent`** — `ScreenshotAnalyzer.cachedAgent` is written without sync.

14. **Fix `go-structure-linter` issues** — Move `coverage.out` to `coverage/`, remove `result` binary, update AGENTS.md freshness.

---

## f) Top 25 Things We Should Get Done Next

| #   | Item                                                               | Impact      | Effort | Category     |
| --- | ------------------------------------------------------------------ | ----------- | ------ | ------------ |
| 1   | Fix `nix flake check` vendorHash                                   | 🔴 Critical | S      | Build        |
| 2   | Fix pre-commit hook (4 failing steps)                              | 🔴 Critical | M      | DevEx        |
| 3   | Resolve LICENSE vs README contradiction                            | 🔴 Critical | S      | Legal        |
| 4   | Add CLI tests (`cmd/vision/main.go`)                               | 🟠 High     | L      | Testing      |
| 5   | Add `internal/cli/helpers.go` tests                                | 🟠 High     | M      | Testing      |
| 6   | Update CHANGELOG.md with 31 commits                                | 🟠 High     | M      | Docs         |
| 7   | Fix WebP validation (check WEBP magic)                             | 🟡 Medium   | S      | Correctness  |
| 8   | Add `io.LimitReader` to `LoadImageFromReader`                      | 🟡 Medium   | S      | Robustness   |
| 9   | Update README.md (fix license, structure)                          | 🟡 Medium   | S      | Docs         |
| 10  | Fix LSP ghost file (`agent_test.go`)                               | 🟡 Medium   | S      | DevEx        |
| 11  | Refactor `AnalyzeStream` to `strings.Builder`                      | 🟢 Low      | S      | Performance  |
| 12  | Add BDD tests for `AnalyzeStructured`                              | 🟢 Low      | S      | Testing      |
| 13  | Add context cancellation tests                                     | 🟡 Medium   | M      | Testing      |
| 14  | Fix `go-structure-linter` issues (coverage.out, binary, AGENTS.md) | 🟡 Medium   | S      | Quality      |
| 15  | Remove deprecated `VisionAgent` alias                              | 🟢 Low      | S      | Cleanup      |
| 16  | Fix `parseFlags()` dead error return                               | 🟢 Low      | S      | Cleanup      |
| 17  | Cache `ScreenshotAnalyzer` agent instance safely                   | 🟢 Low      | S      | Performance  |
| 18  | Refactor `AnalyzeStructured` to use `fantasy.Agent`                | 🟡 Medium   | M      | Architecture |
| 19  | Remove `report/jscpd-report.json` from repo                        | 🟢 Low      | S      | Cleanup      |
| 20  | Enable nix flake package build                                     | 🟡 Medium   | M      | Build        |
| 21  | Migrate justfile commands to flake.nix                             | 🟡 Medium   | L      | Build        |
| 22  | Add `ProviderOptions` passthrough                                  | 🟢 Low      | S      | Feature      |
| 23  | Add `WithModel()` to `ScreenshotAnalyzer`                          | 🟢 Low      | S      | Feature      |
| 24  | Add real provider integration tests                                | 🟡 Medium   | L      | Testing      |
| 25  | Add benchmark tests for hot paths                                  | 🟢 Low      | M      | Performance  |

---

## g) Top #1 Question I Cannot Figure Out Myself 🤔

**Why does the pre-commit hook run the full project lint suite instead of checking only changed files?**

The BuildFlow pre-commit hook runs `go-structure-linter`, `library-policy`, `todo-check`, and `nix-fmt` against the _entire project_ on every commit. This means:

- A commit that only changes `pkg/vision/agent_bdd_test.go` still fails because `AGENTS.md` is 22 days old
- A commit that only adds documentation still fails because `coverage.out` is in the root directory
- A commit that only fixes a bug still fails because the `result` binary exists in the repo

This design defeats the purpose of pre-commit hooks, which should validate _the commit being made_, not the entire project's accumulated debt. The user has to add `--no-verify` to every commit, which trains them to ignore the hook entirely.

**Possible approaches I can see but cannot evaluate without project context:**

1. Fix all underlying issues so the hook passes clean (but then new issues will accumulate)
2. Configure BuildFlow to scope checks to changed files only (does BuildFlow support this?)
3. Remove the failing steps from pre-commit and run them in CI instead
4. Downgrade failing steps from errors to warnings in the pre-commit context

**What is the intended behavior?** Should pre-commit only block commits that introduce _new_ problems, or should it enforce a continuously clean project state?

---

## Changes Since Last Report (May 4 → May 20)

| Change                               | Files                                                                                                             | Impact                                                     |
| ------------------------------------ | ----------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| Added Ginkgo + Gomega BDD test suite | `agent_suite_test.go`, `agent_bdd_test.go`, `screenshot_bdd_test.go`, `image_bdd_test.go`, `validate_bdd_test.go` | 50+ BDD specs covering all user-facing behavior            |
| Strong `MediaType` type              | `image.go`                                                                                                        | Invalid media types caught at compile time                 |
| `NewImageSource` constructor         | `image.go`                                                                                                        | Empty data validation at construction; `ErrEmptyImageData` |
| `Analyzer` interface                 | `vision.go`                                                                                                       | Consumers can mock without depending on concrete `Agent`   |
| Removed duplicate tests              | Deleted `vision_test.go`                                                                                          | Eliminated duplication between table-driven and BDD tests  |
| Fixed all 20 golangci-lint issues    | Multiple                                                                                                          | Lint is now clean (0 issues)                               |
| Updated AGENTS.md                    | `AGENTS.md`                                                                                                       | Documents BDD approach, type model, test organization      |
| Added `ErrEmptyImageData`            | `pkg/errors/errors.go`                                                                                            | Consistent domain error for empty image data               |
| Broke nix flake build                | `flake.nix` (indirect)                                                                                            | vendorHash stale after dep changes                         |

---

## Build & Test Matrix

| Check                     | Status                                         |
| ------------------------- | ---------------------------------------------- |
| `go build ./...`          | ✅ Clean                                       |
| `go test ./...`           | ✅ All pass (50+ specs)                        |
| `go test -race ./...`     | ✅ No races                                    |
| `go vet ./...`            | ✅ Clean                                       |
| `golangci-lint run ./...` | ✅ 0 issues (was 20)                           |
| `go test -cover ./...`    | ✅ 91.8% pkg/vision, 81.8% internal/visionutil |
| `nix flake check`         | ❌ Hash mismatch (vendorHash stale)            |
| BuildFlow pre-commit      | ❌ 4 steps fail (forces `--no-verify`)         |

---

## Dependency Health

| Dependency                    | Version | Status                                                  |
| ----------------------------- | ------- | ------------------------------------------------------- |
| `charm.land/fantasy`          | v0.23.0 | ✅ Current framework                                    |
| `github.com/onsi/ginkgo/v2`   | v2.29.0 | ✅ New — BDD testing                                    |
| `github.com/onsi/gomega`      | v1.41.0 | ✅ New — BDD assertions                                 |
| `github.com/stretchr/testify` | v1.11.1 | ⚠️ Flagged by library-policy (recommend Ginkgo instead) |
| `go.yaml.in/yaml/v3`          | v3.0.4  | ⚠️ Flagged by library-policy (archived repo)            |

---

_Generated by Crush at 2026-05-20T02:11:16+02:00_
