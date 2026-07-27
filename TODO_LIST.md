# TODO List

Short- and mid-term actionable improvement tasks. Each item is bounded and scoped.
For long-term ideas and raw direction, see [ROADMAP.md](ROADMAP.md).
For current feature inventory, see [FEATURES.md](FEATURES.md).

> **Discipline:** When a task is completed, **remove it from this file** and
> record it under `[Unreleased]` in [CHANGELOG.md](CHANGELOG.md). This file is
> for open work only — no `[x]` checkboxes, no "Previously Completed" sections.

---

## Critical — correctness hazards & release blockers

- [ ] **Fix license metadata lie** — `flake.nix:49` still reads `licenses.mit`
  while `LICENSE` is PROPRIETARY. The `[0.2.0]` CHANGELOG claim "corrected to
  `unfree`" was never applied. Change to `licenses.unfree` (or remove the
  field). **Release blocker since 2026-07-23.** (`flake.nix:49`, `LICENSE:1`)
- [ ] **Guard nil `RawResponse` in structured hooks** — `AnalyzeStructured` /
  `AnalyzeStructuredStream` synthesize `&AnalyzeResult{Text, Usage}` with nil
  `RawResponse`. A consumer hook touching `result.RawResponse.Response` panics.
  Either nil-guard the payload or redesign (see "Structured hooks payload"
  below). (`pkg/vision/structured.go:106,226`)
- [ ] **Reconcile `applyModelParams*` duplication** — 2 near-identical helpers
  in `vision.go` + parallel blocks in `structured.go` (4 sites). Extract a
  single generic helper. (`pkg/vision/vision.go:443,465`)
- [ ] **Fix BMP detection vs decoding mismatch** — `MediaTypeBMP` +
  `DetectImageFormat` recognize BMP, but `image.Decode` (used by `ResizeImage`)
  cannot decode BMP without a registered decoder. Register
  `golang.org/x/image/bmp` or document the limitation. (`pkg/vision/preprocess.go`)
- [ ] **Fix `mediaTypeFromExtension` `.bmp` fallback** — relies on
  `mime.TypeByExtension` (system-dependent) and falls back to `MediaTypePNG`
  for unknown extensions. A `.bmp` file likely gets mislabeled. Add an explicit
  `.bmp` → `MediaTypeBMP` case. (`pkg/vision/image.go:183-189`)
- [ ] **Reconcile `Config.MaxRetries` vs `WithRetry[T]`** — two retry systems
  coexist undocumented. Decide: bake `RetryConfig` into `Config` (deprecate
  `MaxRetries`) or keep external and document loudly. Wire the chosen path into
  `AnalyzeBatch` / `AnalyzeConversation`. (`pkg/vision/vision.go:61`,
  `pkg/vision/retry.go`)

## High value — design gaps & completeness

- [ ] **Auto-wire preprocessing** — `ResizeImage` exists but is never called by
  the Agent. Add `Config.Preprocess` (max dimension, quality, format convert)
  applied automatically inside every `Analyze*`, plus
  `ScreenshotAnalyzer.WithMaxDimension`. Add image **compress** (JPEG quality /
  format conversion), not just resize. (`pkg/vision/preprocess.go`)
- [ ] **Solve structured hooks payload properly** — replace the synthesized
  `*AnalyzeResult` hack with either a discriminated `HooksEvent` struct (with a
  `Kind`) or a generic `StructuredHooks[T]` type. Depends on whether a breaking
  `Hooks` change is acceptable (see ROADMAP "Open questions"). Relates to the
  nil-`RawResponse` Critical item above.
- [ ] **Add CLI tests** — `cmd/vision` has zero `_test.go` files. Cover the
  `-structured` branch, provider switch, error-advice mapping, and flag parsing.
  (`cmd/vision/main.go`)
- [ ] **Wire `WithRetry` into `AnalyzeBatch` / `AnalyzeConversation`** —
  currently only the caller can wrap calls in retry.
- [ ] **Add `CostTracker` Agent integration** — standalone type with no Agent
  method. Add `Agent.Cost()` or wire `CostTracker` into `Hooks.OnFinish`
  automatically. (`pkg/vision/cost.go`)
- [ ] **Add `examples/error-handling/main.go`** — consumer-facing example
  showing the `errors.AsType[*vision.ModelError](err)` → switch-on-`Kind`
  pattern.

## Config & tooling hygiene

- [ ] **Root-cause depguard `$module`** — the hardcoded module path works but is
  fragile. Check golangci-lint v2 docs/issues for the correct modern syntax;
  restore the variable form if possible, else add an explanatory comment in
  `.golangci.yaml`. (`.golangci.yaml`)
- [ ] **Tighten `nolintlint`** — enable `require-explanation: true` and
  `allow-no-extra-linter: false` so dead/anonymous `//nolint:` directives fail
  the build (prevents the 6 dead `legacyerrors` directives from recurring).
  (`.golangci.yaml`)
- [ ] **Run `golangci-lint config verify`** — built-in config validator, never
  invoked. Add to CI once CI exists.
- [ ] **Run `nix flake check`** — the canonical quality gate (per AGENTS.md).
  Confirm it passes; fix anything it flags.
- [ ] **Add CI workflow** — no `.github/workflows/` exists. Add build, vet,
  test (with `-race`), lint, and coverage gate (≥70%) on push/PR.

## Testing gaps

- [ ] **BDD (Ginkgo) specs for error classification** — currently testify
  table-driven only; project convention is BDD for user-facing behavior.
  (`pkg/vision/error_classification_test.go`)
- [ ] **`AnalyzeBatch` classified-error test** — no test verifying per-image
  errors in `BatchResult.Err` are classified `*ModelError`s.
- [ ] **Remove dead `errTestNoop` sentinel** — defined but never used.
  (`pkg/errors/model_test.go:22`)
- [ ] **Replace `wrapNoop` with real chain traversal** — current `wrapNoop`
  returns its argument unchanged, testing nothing about error-chain traversal.
  Use `fmt.Errorf("wrapped: %w", err)`. (`pkg/errors/model_test.go:316-320`)

## Error-kind refinements

- [ ] **Consider `KindNotImplemented`** for HTTP 501 (currently
  `KindServerError` / retryable, but 501 is not really retryable).
- [ ] **Consider `KindServiceUnavailable`** for HTTP 503 (currently lumped into
  `KindServerError`).
- [ ] **Consider `KindContentFilter`** for provider content-policy rejections
  (some providers return 400 with specific messages).

## Release mechanics

- [ ] **Create GitHub Release for v0.2.0** — tag is pushed but no release notes
  exist (`gh release view v0.2.0` → not found).
- [ ] **Resolve tag anomaly** — `v0.2.1` and `v0.3.0` both point to commit
  `d5dda4b` (a pre-v0.2.0 test-formatting commit). Decide: delete and re-tag,
  or supersede with a real `v0.3.0` once `[Unreleased]` work is tagged.
  **Destructive — requires explicit user approval (force-push / tag deletion).**
- [ ] **Update `CONTRIBUTING.md`** — still references bare `go test` /
  `golangci-lint`, not the flake commands (`nix run .#test`, `nix run .#lint`).

## Documentation

- [ ] **Update `docs/DOMAIN_LANGUAGE.md`** — add terms for retry (`RetryConfig`,
  `WithRetry`), cost (`CostTracker`, `Usage`), and preprocessing
  (`ResizeImage`, `PreprocessConfig`).
