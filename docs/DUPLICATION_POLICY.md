# Code Duplication Policy

This document records the project's duplication-elimination work and the helpers
that keep the codebase DRY. It is the **single source of truth** for duplication
decisions in this repo.

## Current State

**Verified with `art-dupl --type-aware -t 1` (v0.6.1, 2026-08-17): 0 actionable
clone groups.** The scan now covers `internal/reviewd` and
`cmd/visionreviewd` (added with the daemon). Raw findings: 254 groups total,
of which 223 are non-actionable and 30 suppressed by filters; the single
actionable pair after the extractions below is the 6-line `once`/`run`
prologue (see "Patterns Below Scan Scope").

Test files (`*_test.go`, `*_bdd_test.go`) are auto-excluded by art-dupl;
interface-required signatures and table-driven test rows are inherently
irreducible and never appear in the scan.

## Helpers in Place

### Production code (`pkg/`, `internal/`, `cmd/`)

| Helper                                              | Location                        | Purpose                                                                                                                                                                                                                     |
| --------------------------------------------------- | ------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `requireImages`                                     | `pkg/vision/vision.go`          | Single source for `ErrNoImages` guard                                                                                                                                                                                       |
| `validateAnalyzeInput`                              | `pkg/vision/vision.go`          | Enforces `prompt != ""` + `requireImages` in one call                                                                                                                                                                       |
| `preparedRequest` + `prepare`                       | `pkg/vision/vision.go`          | Shared prologue: validate → preprocess → fireStart → timeout → toFileParts                                                                                                                                                  |
| `withPrepared[T]`                                   | `pkg/vision/vision.go`          | Generic higher-order wrapper that owns the `prepare() + if err + defer cancel()` idiom once, used by all 6 analysis methods (4 `Analyze*` + 2 `AnalyzeStructured*`). Eliminated the last 5-site `if err != nil` clone group |
| `finishResult`                                      | `pkg/vision/vision.go`          | Shared epilogue: builds `AnalyzeResult`, fires `fireFinish`                                                                                                                                                                 |
| `buildObjectCall[T]`                                | `pkg/vision/structured.go`      | Shared `fantasy.ObjectCall` construction (18 lines → 1 call)                                                                                                                                                                |
| `invalidate`                                        | `pkg/vision/screenshot.go`      | Cache invalidation for every `ScreenshotAnalyzer.With*` builder (11 sites → 1)                                                                                                                                              |
| `optionalParams`                                    | `pkg/vision/vision.go`          | Single source for optional model params fed to all 4 call sites                                                                                                                                                             |
| `imageSignature` (named type)                       | `pkg/vision/validate.go`        | Named struct replaces double anonymous declaration                                                                                                                                                                          |
| `jsonOutput` / `jsonUsage` (named types)            | `cmd/vision/main.go`            | Named structs replace inline anonymous struct                                                                                                                                                                               |
| `newProviderFromEnv` / `wrapProvider`               | `cmd/vision/main.go`            | API-key-from-env + provider factory + error wrapping                                                                                                                                                                        |
| `createOpenAIProvider` / `createOpenRouterProvider` | `cmd/vision/main.go`            | Named factories for provider constructors                                                                                                                                                                                   |
| `parseConfigFlag`                                   | `cmd/visionreviewd/commands.go` | Shared `-config` flag parse + config load for all four config-taking daemon commands (once/run/replay/doctor)                                                                                                               |
| `openConfiguredPipeline` / `closeStore`             | `cmd/visionreviewd/commands.go` | Configured pipeline + store opening and the deferred close-error report shared by `once` and `run`                                                                                                                          |
| `ReviewsDirPermission` / `ReviewsFilePermission`    | `internal/reviewd/writer.go`    | Exported once, used by both the Writer and the doctor probes, so the probe modes can never drift from the write modes                                                                                                       |

### CLI helpers (`internal/cli/`)

| Helper                                                          | Location                  | Purpose                                                                                                                                                                             |
| --------------------------------------------------------------- | ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ExitOnError` / `RequireArgc` / `RequireEnvVar` / `PrintResult` | `internal/cli/helpers.go` | Shared CLI primitives                                                                                                                                                               |
| `NewAgent`                                                      | `internal/cli/helpers.go` | Agent from model + system prompt + optional temperature                                                                                                                             |
| `NewOpenAIModel`                                                | `internal/cli/openai.go`  | OpenAI provider+model from env (exits on error)                                                                                                                                     |
| `LoadImageArg`                                                  | `internal/cli/helpers.go` | Load image from `os.Args[1]` (exits on error)                                                                                                                                       |
| `NewCLIContext`                                                 | `internal/cli/helpers.go` | Arg validation + background context + default gpt-4o model. Used by examples that need custom `Config`                                                                              |
| `NewAgentFromArgs`                                              | `internal/cli/helpers.go` | One-line bootstrap: `NewCLIContext` + `NewAgent` + `ExitOnError`. Collapsed the 5-site example prologue to a single call                                                            |
| `AnalyzeAndPrint`                                               | `internal/cli/helpers.go` | One-call workflow: `LoadImageArg` → `Analyze` → `ExitOnError` → `PrintResult`. Eliminated the `LoadImageArg + Analyze + ExitOnError + PrintResult` clone run across simple examples |

### Shared example schema (`examples/internal/`)

| Helper                                 | Location                                 | Purpose                                                                                                                                           |
| -------------------------------------- | ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `uireview.UIReview` / `uireview.Issue` | `examples/internal/uireview/uireview.go` | Canonical structured-output schema shared by `examples/structured` and `examples/structured-stream`. Eliminated the duplicated struct definitions |

## Patterns Below Scan Scope (irreducible by nature)

These patterns are inherently duplicated but never appear in `art-dupl` output
because they are either auto-excluded (test files) or structurally required by
Go's type system:

| Pattern                                                                                           | Why it's irreducible                                                                                                                                   |
| ------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `Analyzer` interface method signatures on `*Agent` and `*ScreenshotAnalyzer`                      | Go requires identical signatures for interface satisfaction                                                                                            |
| `*_test.go` / `*_bdd_test.go` assertion shapes (`Expect(...).To(...)`)                            | Test files auto-excluded; each spec is a self-contained specification                                                                                  |
| Table-driven test rows `{"png", []byte{...}, true}`                                               | Data rows, not code duplication                                                                                                                        |
| Mock model method signatures (`Generate`, `Stream`, `GenerateObject`, `StreamObject`)             | Required by `fantasy.LanguageModel` interface                                                                                                          |
| `type testReview struct{...}` in `internal/visionutil/helpers_test.go`                            | Cross-package test fixture; cannot be shared without a third test-helper package                                                                       |
| `once`/`run` command prologue (6 lines: `openConfiguredPipeline` + `if !ok` + `defer closeStore`) | Two commands genuinely perform the same opening; a closure-passing abstraction would hide control flow for zero duplication gain (accepted 2026-08-17) |
