# Vision Review Agent — Comprehensive Status Report

**Date:** 2026-05-04 21:44 CEST
**Branch:** master (up to date with origin)
**Commits:** 25 total
**Go Version:** 1.26.2 (in go.mod)
**LOC:** 2,376 lines of Go code across 21 files
**Test Coverage:** 43.8% overall | 94.2% `pkg/vision` | 81.8% `internal/visionutil` | 0% CLI/examples

---

## Executive Summary

Vision Review Agent is a Go SDK for building AI agents with vision capabilities, built on `charm.land/fantasy v0.23.0`. The core library (`pkg/vision`) is solid with 94.2% test coverage and a clean architecture. The project has undergone significant refactoring since the last status report (April 27), including extraction of shared CLI helpers, test assertion helpers, and adding `MaxOutputTokens`/`Temperature` passthrough to the fantasy agent calls. However, there are functional bugs, documentation drift, lint issues, and a stale changelog that need attention before any public release.

---

## a) FULLY DONE ✅

| Item                                | Details                                                                                                            |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| Core SDK (`pkg/vision`)             | `Agent`, `Config`, `Analyze()`, `AnalyzeStream()`, `AnalyzeStructured[T]()` — all working                          |
| `ScreenshotAnalyzer` fluent builder | Complete with `WithSystemPrompt`, `WithMaxOutputTokens`, `WithTemperature`, `WithMaxRetries`, `WithRequestTimeout` |
| Image loading                       | `LoadImageFromFile`, `LoadImageFromReader` with media type detection                                               |
| Image validation                    | Magic byte detection for PNG, JPEG, GIF, WebP, BMP — 100% test coverage                                            |
| Structured output                   | Generic `AnalyzeStructured[T]()` with JSON schema — 82.6% coverage                                                 |
| Centralized errors                  | `pkg/errors/` with 6 sentinel errors, re-exported from `pkg/vision/` for backward compat                           |
| Table-driven tests                  | All test files use table-driven pattern consistently                                                               |
| Test assertion helpers              | Extracted shared helpers in `mock_test.go` (`AssertErr`, `AssertEq`, etc.)                                         |
| Build system                        | `flake.nix` + `justfile` + `go build` — all green                                                                  |
| CLI binary                          | `cmd/vision/main.go` — supports OpenAI/OpenRouter, streaming, JSON output, timeout                                 |
| Examples                            | 3 working examples (openai, openrouter, structured)                                                                |
| Internal helpers                    | `internal/visionutil/` (prompt building, JSON round-trip) — 81.8% coverage                                         |
| Internal CLI helpers                | `internal/cli/helpers.go` (`ExitOnError`, `RequireArgc`, `RequireEnvVar`, `PrintResult`, `NewAgent`)               |
| Race detector                       | All tests pass with `-race` flag                                                                                   |
| `go vet`                            | Clean — zero warnings                                                                                              |
| Git workflow                        | Git Town configured, clean branching                                                                               |

---

## b) PARTIALLY DONE ⚠️

| Item                                        | Status                                                | What's Left                                                                                                                                 |
| ------------------------------------------- | ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `MaxOutputTokens`/`Temperature` passthrough | Code added in `vision.go` uncommitted diff            | **Bug: sends `&0` when not configured** — always sets pointers even for zero values, which may truncate output at providers                 |
| Test coverage                               | 94.2% on `pkg/vision`, 81.8% on `internal/visionutil` | **0% on `cmd/vision`, `internal/cli`, all examples** — overall 43.8%                                                                        |
| Lint compliance                             | 20 issues remaining                                   | 8× `copyloopvar`, 6× `paralleltest`, 2× `staticcheck`, 1× each `revive`/`testifylint`/`unparam`/`wrapcheck`                                 |
| Documentation                               | README mostly current                                 | Missing `internal/cli/` in structure, Anthropic mentioned but not supported, LICENSE contradiction (README says MIT, file says PROPRIETARY) |
| CHANGELOG.md                                | Exists                                                | Only lists "Initial release 0.1.0" — 25 commits of work untracked                                                                           |
| AGENTS.md                                   | Mostly current                                        | Missing `internal/cli/` and `report/` directory in architecture tree                                                                        |
| `goccy/go-yaml` advisory                    | Researched                                            | Determined to be a false positive — transitive dependency 3 levels deep, not actionable from this project                                   |

---

## c) NOT STARTED ❌

| Item                                                | Priority     | Notes                                                     |
| --------------------------------------------------- | ------------ | --------------------------------------------------------- |
| CLI tests (`cmd/vision/main.go`)                    | High         | 0% coverage, 263 lines untested                           |
| `internal/cli/helpers.go` tests                     | High         | 0% coverage, uses `os.Exit` (untestable as-is)            |
| Fix `MaxOutputTokens=0` sent as `&0` bug            | **Critical** | Functional bug in uncommitted code                        |
| WebP validation false positive fix                  | Medium       | Only checks RIFF header, not WEBP magic                   |
| `io.ReadAll` size limit in `LoadImageFromReader`    | Medium       | No protection against OOM on huge files                   |
| `AnalyzeStructured` bypassing `fantasy.Agent` layer | Medium       | Duplicates system prompt, skips agent middleware          |
| `ScreenshotAnalyzer` agent caching                  | Low          | New `Agent` allocated per call                            |
| Remove deprecated `VisionAgent` alias               | Low          | Still exported, causes `revive` stutter warning           |
| `parseFlags()` dead error return                    | Low          | Always returns `nil`                                      |
| Remove stale `report/jscpd-report.json`             | Low          | Artifact in repo                                          |
| Version bump / CHANGELOG update                     | Medium       | 0.1.0 from initial release, 25 commits of changes since   |
| Nix flake package build                             | Medium       | Currently commented out in `flake.nix`                    |
| `AnalyzeStream` string concatenation                | Low          | `fullText += text` is O(n²), should use `strings.Builder` |
| Consistent test assertion style                     | Low          | Mix of stdlib, testify, custom helpers                    |

---

## d) TOTALLY FUCKED UP 💥

| Item                                              | Severity    | Details                                                                                                                                                                                                                                                                                                           |
| ------------------------------------------------- | ----------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **`MaxOutputTokens=0` sent as `&0` to providers** | 🔴 Critical | In uncommitted `vision.go` diff: `MaxOutputTokens: va.config.MaxOutputTokens` always sets the pointer. When config has zero (default), `&0` is sent to the provider. Some providers may interpret this as "generate 0 tokens" = empty response. Must only set when explicitly configured. Same for `Temperature`. |
| **LICENSE vs README contradiction**               | 🔴 Legal    | `LICENSE` file contains PROPRIETARY text. `README.md` line 179 says "MIT License". These contradict — one of them is wrong.                                                                                                                                                                                       |
| **CHANGELOG 25 commits stale**                    | 🟠 High     | `[0.1.0]` says "Initial release" from 2026-01-01. 25 commits of refactoring, features, and fixes have zero changelog entries. Anyone looking at the changelog has no idea what changed.                                                                                                                           |
| **`golangci-lint` 20 issues unfixed**             | 🟡 Medium   | 8 `copyloopvar` (trivial fix: delete `tc := tc` lines), 6 `paralleltest` (add `t.Parallel()`), 2 `staticcheck` (nil pointer risk in test), 4 misc. All mechanical fixes.                                                                                                                                          |

---

## e) WHAT WE SHOULD IMPROVE 🏗️

1. **Fix the `MaxOutputTokens`/`Temperature` bug** — only send pointers when values are explicitly set (> 0 for tokens, any value for temperature if set). This is a real functional bug that could break production usage.

2. **Fix the LICENSE contradiction** — decide: is this PROPRIETARY or MIT? Update whichever is wrong. This is a legal issue.

3. **Update CHANGELOG.md** — 25 commits of work should be reflected. Use `git log --oneline` to reconstruct the timeline.

4. **Fix all 20 lint issues** — they're all mechanical. `copyloopvar`: delete 8 lines. `paralleltest`: add 6 calls. `staticcheck`: fix nil checks. `wrapcheck`: wrap error. `unparam`: remove dead error return.

5. **Add CLI tests** — `cmd/vision/main.go` is 263 lines with 0% coverage. At minimum test `parseFlags`, `buildConfig`, `parseTimeout`, `loadImages`, `printText`, `printJSON`, `createProvider`.

6. **Make `internal/cli` testable** — `ExitOnError` calls `os.Exit()` directly. Refactor to return errors instead, or use `os.Exit` wrapper that can be intercepted in tests.

7. **Fix WebP validation** — check bytes 8-11 are `WEBP` in addition to `RIFF` header. Current code accepts any RIFF file (WAV, AVI, etc.) as valid WebP.

8. **Add `io.LimitReader` to `LoadImageFromReader`** — prevent OOM from unbounded reads. Suggest 50MB default limit.

9. **Refactor `AnalyzeStructured` to use `fantasy.Agent`** — currently bypasses the agent layer and duplicates system prompt handling. Inconsistent with `Analyze()`/`AnalyzeStream()`.

10. **Consistent test assertion style** — pick one: stdlib `testing` or `testify`. Currently mixed. `errors_test.go` uses `testify/assert`, everything else uses custom helpers or stdlib.

11. **Remove stale `report/jscpd-report.json`** — artifact that doesn't belong in the repo. Add to `.gitignore`.

12. **Fix `.gitignore` formatting** — lines 31-36 missing `#` comment prefixes.

13. **Update AGENTS.md** — add `internal/cli/` and `report/` to architecture tree. Note fantasy version.

14. **Migrate justfile → flake.nix** — per AGENTS.md global policy: "justfile is deprecated, should be migrated to flake.nix". Currently both exist.

15. **Fix `go.mod` Go version** — `go 1.26.2` in go.mod; the previous status report flagged this as potentially wrong. Verify and align.

---

## f) Top 25 Things We Should Get Done Next

| #  | Item                                                               | Impact      | Effort | Category     |
| -- | ------------------------------------------------------------------ | ----------- | ------ | ------------ |
| 1  | Fix `MaxOutputTokens=0`/`Temperature` pointer bug                  | 🔴 Critical | S      | Bug          |
| 2  | Resolve LICENSE vs README contradiction                            | 🔴 Critical | S      | Legal        |
| 3  | Fix all 20 golangci-lint issues                                    | 🟠 High     | S      | Quality      |
| 4  | Update CHANGELOG.md with all 25 commits                            | 🟠 High     | M      | Docs         |
| 5  | Add tests for `cmd/vision/main.go`                                 | 🟠 High     | L      | Testing      |
| 6  | Add tests for `internal/cli/helpers.go`                            | 🟠 High     | M      | Testing      |
| 7  | Fix WebP validation (check WEBP magic bytes 8-11)                  | 🟡 Medium   | S      | Correctness  |
| 8  | Add `io.LimitReader` to `LoadImageFromReader`                      | 🟡 Medium   | S      | Robustness   |
| 9  | Update README.md (remove Anthropic, fix structure, fix license)    | 🟡 Medium   | S      | Docs         |
| 10 | Update AGENTS.md (add `internal/cli/`, `report/`, fantasy version) | 🟡 Medium   | S      | Docs         |
| 11 | Remove deprecated `VisionAgent` alias                              | 🟢 Low      | S      | Cleanup      |
| 12 | Fix `parseFlags()` dead error return                               | 🟢 Low      | S      | Cleanup      |
| 13 | Refactor `AnalyzeStructured` to use `fantasy.Agent` layer          | 🟡 Medium   | M      | Architecture |
| 14 | Cache `ScreenshotAnalyzer` agent instance                          | 🟢 Low      | S      | Performance  |
| 15 | Use `strings.Builder` in `AnalyzeStream`                           | 🟢 Low      | S      | Performance  |
| 16 | Consistent test assertion style (pick testify or stdlib)           | 🟢 Low      | M      | Style        |
| 17 | Remove `report/jscpd-report.json` from repo                        | 🟢 Low      | S      | Cleanup      |
| 18 | Fix `.gitignore` formatting (missing `#` on lines 31-36)           | 🟢 Low      | S      | Cleanup      |
| 19 | Commit the uncommitted `vision.go` diff (with bug fix)             | 🔴 Critical | S      | Git          |
| 20 | Enable nix flake package build                                     | 🟡 Medium   | M      | Build        |
| 21 | Migrate justfile commands to flake.nix                             | 🟡 Medium   | L      | Build        |
| 22 | Add `ProviderOptions` passthrough in `toFileParts`                 | 🟢 Low      | S      | Feature      |
| 23 | Add `WithModel()` to `ScreenshotAnalyzer` builder                  | 🟢 Low      | S      | Feature      |
| 24 | Fix `go.mod` Go version alignment                                  | 🟢 Low      | S      | Config       |
| 25 | Add integration test with real provider (mocked)                   | 🟡 Medium   | L      | Testing      |

---

## g) Top #1 Question I Cannot Figure Out Myself 🤔

**Is this project PROPRIETARY or MIT-licensed?**

The `LICENSE` file contains proprietary text (all rights reserved). The `README.md` says "MIT License" and shows the MIT badge. The `PUBLIC_OR_PRIVATE.md` document discusses open-sourcing but hasn't reached a final decision.

This matters because:

- If **proprietary**: README is wrong and misleading — fix the README
- If **MIT**: LICENSE file is wrong — fix the LICENSE file
- This affects whether we should publish to pkg.go.dev, how examples can be used, and whether contributors need CLAs

The license status also blocks the `PUBLIC_OR_PRIVATE.md` decision — we can't open-source without a clear, correct license.

---

## Uncommitted Changes

```
pkg/vision/vision.go | 20 +++++++++++++-------
1 file changed, 13 insertions(+), 7 deletions(-)
```

**Diff summary:** Adds `MaxOutputTokens` and `Temperature` passthrough to `fantasy.AgentCall` and `fantasy.AgentStreamCall`, adds `ProviderOptions: nil` to `toFileParts`, and a minor comment formatting fix on the deprecated `VisionAgent` alias.

⚠️ **This diff contains the `MaxOutputTokens=0` bug and must be fixed before committing.**

---

## Build & Test Matrix

| Check                     | Status                        |
| ------------------------- | ----------------------------- |
| `go build ./...`          | ✅ Clean                      |
| `go test ./...`           | ✅ All pass                   |
| `go test -race ./...`     | ✅ No races                   |
| `go vet ./...`            | ✅ Clean                      |
| `golangci-lint run ./...` | ⚠️ 20 issues                   |
| `go test -cover ./...`    | ⚠️ 43.8% overall               |
| Coverage threshold (70%)  | ✅ Passes (pkg+internal only) |

---

## Dependency Health

| Dependency                    | Version            | Status                                 |
| ----------------------------- | ------------------ | -------------------------------------- |
| `charm.land/fantasy`          | v0.23.0            | ✅ Current framework                   |
| `github.com/stretchr/testify` | v1.11.1            | ✅ Current                             |
| `github.com/goccy/go-yaml`    | v1.19.2 (indirect) | ⚠️ Transitive only, not actionable here |
| `gopkg.in/yaml.v3`            | v3.0.1 (indirect)  | ✅ Standard library adjacent           |

---

_Generated by Crush at 2026-05-04T21:44:58+02:00_
