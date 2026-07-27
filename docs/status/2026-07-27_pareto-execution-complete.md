# Status — Pareto Post-TODO Execution: COMPLETE

**Date:** 2026-07-27
**Source plan:** `docs/planning/2026-07-27_21-18_pareto-post-todo-execution-plan.md`
**Outcome:** All 7 epics (34 subtasks) executed and verified. E2.5 (tag anomaly)
remains blocked on a user decision (documented, not destructive).

---

## Verification gates (all green)

| Gate | Result |
| --- | --- |
| `go build ./...` | OK |
| `go vet ./...` | OK |
| `go test -race -count=1 ./...` | PASS (~3.6s wall) |
| `golangci-lint run ./...` | 0 issues |
| `golangci-lint config verify` | OK |
| `gofumpt -l .` | CLEAN |
| `nix flake check` | **all checks passed!** |
| `go mod verify` + `go mod tidy` | clean, no diff |
| Coverage | **88.5%** (pkg/errors 96.6%, pkg/vision 87.7%; gate is 70%) |

---

## What changed per epic

### E1 — Test suite speedup (corrective finding)
**The plan's premise was inverted.** It claimed the suite took 280s and proposed
adding `MaxRetries: 1` everywhere. In reality the suite was already ~11s because
`MaxRetries` defaults to 0 (which disables fantasy's HTTP retry). The actual
slowness lived in the **3 vision-layer retry tests** that had *deliberately* set
`MaxRetries: 1` — that single setting re-enabled fantasy's ~5s backoff per
retryable mock call.

**Fix:** removed `MaxRetries: 1` from the retry tests (they test vision-layer
retry, not fantasy HTTP-layer retry) and tightened their assertions to exact
`Equal()` counts with documented arithmetic. Result: **11s → 3.6s** full race
suite, deterministic counts.

### E2 — Release mechanics
`nix flake check` passes (fixed 3 `meta.description` app warnings). `go mod
verify`/`tidy` clean. Annotated the `[0.2.0]` CHANGELOG license line as
retroactively false (non-destructive). Tag-anomaly recommendation written to
ROADMAP open-questions (v0.2.1/v0.3.0 both point to a pre-v0.2.0 commit;
destructive resolution needs user approval).

### E3 — PreprocessConfig honesty fix
`PreprocessConfig.JPEGQuality` existed but did nothing. Wired it end-to-end:
added `ResizeImageWithQuality`, `CompressImage` (re-encode without resize; PNG
preserves format), and a shared `encodeImage` helper used by both resize and
compress. PNG output now uses `BestCompression`. Tests verify smaller bytes at
lower quality and format preservation.

### E4 — Docs sync
Rewrote `README.md` (removed `just`/`vision-cli` lies; added Retry, Preprocess,
CompressImage, CostTracker, all 14 ErrorKinds, correct `errors.AsType` signature,
real Hooks signatures). Updated `FEATURES.md` (moved resolved items to DONE).
Cross-linked `docs/DOMAIN_LANGUAGE.md`. Verified all code snippets against the
real API.

### E5 — Test coverage for new code (8 subtasks)
`mediaTypeFromExtension` table; BMP decode→resize roundtrip (hand-rolled a
minimal valid BMP since Go has no BMP encoder); `PreprocessImage` nil/zero
passthrough; `Config.Preprocess` end-to-end in `Analyze` + `AnalyzeStructured`
(via an opt-in capturing mock field); `NewAgentWithCostTracker` nil-RawResponse
contract; `contentFilter` signal detection; 501/503/contentFilter via full
`Analyze`; and a **fixed** `AnalyzeBatch` mixed success+error test (the prior
version was misnamed — both images failed).

### E6 — CI hardening
Added a `go mod tidy` diff check, a `golangci-lint config verify` step, and a
dedicated `nix-flake-check` job (cachix/install-nix-action) to
`.github/workflows/ci.yml`.

### E7 — CLI testability
Refactored `parseFlags()` → `parseFlags(fs *flag.FlagSet, args []string)`
returning errors instead of calling `os.Exit`; version/no-args decisions surface
as `cfg.showVersion` / `cfg.args`. `loadImages` now takes the args slice. Added
5 tests: defaults, all flags, `-version`, missing args, unknown flag.

---

## Files touched (this session)

**Production code:** `pkg/vision/preprocess.go` (rewrite: quality wiring +
`CompressImage`/`ResizeImageWithQuality`/`encodeImage`), `cmd/vision/main.go`
(`parseFlags` FlagSet refactor), `flake.nix` (app `meta.description`).

**Tests:** `pkg/vision/preprocess_test.go`, `pkg/vision/image_test.go` (new),
`pkg/vision/mock_test.go` (capturing field), `pkg/vision/retry_test.go`,
`pkg/vision/error_classification_test.go`, `pkg/errors/model_test.go`,
`cmd/vision/main_test.go`.

**Docs/config:** `README.md`, `FEATURES.md`, `CHANGELOG.md`, `AGENTS.md`,
`ROADMAP.md`, `.github/workflows/ci.yml`, the planning doc (status → EXECUTED).

---

## Deferred / blocked (not Verschlimmbessern)

- **E2.5 tag anomaly** — blocked on user (destructive: delete/move v0.2.1,
  v0.3.0). Recommendation in ROADMAP.
- **Structured hooks redesign** (`HooksEvent`) — breaking change, no demand;
  ROADMAP.
- **Streaming auto-retry** — deliberately excluded (ambiguous delta semantics).
- New ErrorKinds (529, 402), `RetryAfter`, OTel, provider failover, caching,
  `catwalk` — all ROADMAP, per the plan's risk analysis.

## Open questions for the user (unchanged, non-blocking)

1. Tag anomaly: delete + re-tag, or supersede with a new tag?
2. Breaking `Hooks` change acceptable in the next minor?
3. Should streaming methods auto-retry?
