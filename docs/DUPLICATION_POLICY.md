# Code Duplication Policy

This document records the project's duplication-elimination work and the helpers
that keep the codebase DRY. It is the **single source of truth** for duplication
decisions in this repo.

## Current State

**Verified with `art-dupl --sort total-tokens -t 1 --type-aware` over the whole
repo (v0.6.x, 2026-09-22): 7 clone groups, all accepted with rationale below.**
The 2026-09-22 pass eliminated 3 harmful groups: the four hand-rolled
sorted-map-keys helpers (replaced by stdlib `slices.Sorted(maps.Keys(m))`,
plus `slices.SortFunc` for the struct sort), the 4-site payload-decode error
contract in `pkg/vision/a2ui/messages.go` (extracted into `decodePayload`),
and the two prompt-assembly skeletons in `internal/reviewd/prompts.go`
(extracted into `buildPrompt` + instruction constants).

Earlier: `art-dupl --type-aware -t 1` (v0.6.1, 2026-08-17) over
`internal/reviewd` + `cmd/visionreviewd`: 0 actionable groups (254 raw, 223
non-actionable, 30 suppressed); the single actionable pair after the
extractions below was the 6-line `once`/`run` prologue (see "Patterns Below
Scan Scope").

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
| `decodePayload`                                     | `pkg/vision/a2ui/messages.go`   | Single source for the `ErrMalformedMessage` + `decode <kind> payload` contract shared by all four message-kind `UnmarshalJSON` methods                                                                                      |
| `buildPrompt` + `reviewInstructions`/`compareInstructions` | `internal/reviewd/prompts.go` | Shared prompt assembly (view context → instructions → `scoreRules`); the per-persona sections live as pinned string constants                                                                                          |
| `newConfigFlagSet`                                  | `cmd/visionreviewd/commands.go` | Flag set carrying the `-config` flag every daemon command shares, so its default and description are defined in exactly one place                                                                                          |
| `slices.Sorted(maps.Keys(m))` (stdlib, not a helper) | call sites in `discover.go`/`pipeline.go`/`replay.go` | Replaced the four per-map sorted-keys helpers; Go 1.23+ makes them redundant |

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

## Accepted clone groups (2026-09-22 full-repo pass)

The 7 groups `art-dupl -t 1 --type-aware` still reports are all deliberate:

| Group | Why it stays |
| ----- | ------------ |
| `vision.go` `params := optionalParams()` + 6 field copies in `buildAgentCall` / `buildAgentStreamCall` (and the same shape in `buildObjectCall`) | Three DISTINCT `fantasy` call types share field names but no interface; Go cannot copy same-named fields across unrelated struct types without reflection. The source (`optionalModelParams`) is already single-sourced, so a new parameter lands in the params struct once and in N dumb assignment blocks. |
| `component.go` / `image.go` / `preprocess.go` 5-line `if err != nil` blocks | Unrelated domains (child-list JSON encode, base64 decode, image decode) behind a universal Go idiom — scan noise at `-t 1`. |
| `examples/a2ui` + `examples/structured` `cli.NewAgentFromArgs(2, "...")` | The clone IS the shared bootstrap helper; the examples differ by prompt and schema on purpose. |
| `config.go` + `store.go` error-wrap blocks | Unrelated operations (config JSON encode vs journal read); idiomatic wrap per site. |
| `generate.go` `applyDefaults` + `surface.go` `Compile` zero-checks | Two different types (`GenerateOptions` vs `SurfaceSpec`); the default VALUES (`defaultSurfaceID`, `DefaultCatalogID`) are already single-sourced constants. |
| `commands.go` `newConfigFlagSet("compare"/"events", stderr)` call lines | The clone is the shared-helper call itself (same class as `NewAgentFromArgs`); the per-command flags deliberately stay local. |
| `commands.go` `if flagSet.NArg() != 1 { usage; return exitUsage }` ×3 | Usage lines are per-command data; a `requireArgCount` helper would force callers through a code-return dance that is longer than the original (same precedent as the `once`/`run` prologue). |

## pkg/vision/a2ui (scanned 2026-08-18, post-builders)

`art-dupl --type-aware -t 1` over the package: **0 actionable clone groups**
after extracting `urlDescriptionProps` (the shared url+optional-description
property set of NewImage/NewAudioPlayer).

Judgment calls:

- **Builder similarity is intentional** — the 18 `New<Kind>` constructors
  share a shape (id → Component with Kind + props) by design: each is a
  two-to-six-line typed wrapper whose value is its exact signature. Further
  extraction (a generic `newComponent(kind, props)` helper) would erase the
  type-level documentation the builders exist to provide.
- **Wire marshal/unmarshal pairs** (Component, ChildList, the four message
  kinds) look structurally similar but encode opposite directions with
  kind-specific envelope handling; merging them would trade clarity for a
  false DRY win.
- Test files remain below scan scope by policy (table-driven tests repeat
  shapes deliberately).
