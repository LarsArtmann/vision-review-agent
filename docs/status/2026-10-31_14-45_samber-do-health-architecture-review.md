# samber/do v2 + Health Checks — Deep Architecture Review

**Date:** 2026-10-31 · **Scope:** whole repo · **Verdict: premise fails, architecture is sound, one real gap existed and is now fixed**

---

## 1. The premise, checked honestly

> "Are we using samber/do v2 in combination with Health Checks superbly?!"

**No — because samber/do is not used at all.** Verified 2026-10-31:

| Where                 | Finding                                                                                               |
| --------------------- | ----------------------------------------------------------------------------------------------------- |
| `go.mod`              | no `github.com/samber/do` requirement                                                                 |
| `go.sum`              | only `github.com/samber/lo v1.53.0` (a generics utility lib, transitively via `go-cqrs-lite/otel/v4`) |
| `grep -r 'samber/do'` | zero `.go` hits                                                                                       |
| `go mod graph`        | `go-cqrs-lite/otel/v4 → samber/lo` only — `samber/do` never enters the graph                          |

`samber/lo` is not samber/do. There is no DI container, no `do.New()`, no `do.Healthchecker`, nothing to audit against DO-1 → DO-6 directly.

## 2. What the project uses instead — and how it scores

The daemon wires its services with **manual constructor injection**, and it does it well:

| samber/do canonical pattern                  | This repo's equivalent                                                                                 | Score        |
| -------------------------------------------- | ------------------------------------------------------------------------------------------------------ | ------------ |
| Composition root                             | `openPipeline` (cmd/visionreviewd/commands.go:58): builds reviewer → store → blob store → writer       | ✅           |
| Lazy singletons                              | N/A (4 services, one process, one lifetime) — plain structs are the right weight                       | ✅           |
| Validate wiring at construction              | `NewDaemon` / `NewPipeline` reject nil deps with exported sentinels (`ErrNoPipeline`, `ErrNoStore`, …) | ✅ exemplary |
| Program to interfaces                        | `PassRunner` (daemon.go:21), `fantasy.LanguageModel` (provider.go:16), `vision.Analyzer`               | ✅           |
| `do.ShutdownerWithError` on resource holders | `Store.Close` (store.go:224); every command defers `closeStore` and reports close errors               | ✅           |
| `do.Healthchecker` on the model endpoint     | **was missing at runtime** — doctor CLI only → **fixed this pass** (see §4)                            | ✅ now       |
| DO-2 analog: no leaked resources             | every `OpenStore` call site closes: commands.go:77 (pipeline-build failure), 186, 266, 419, 513, 577   | ✅           |
| DO-4 analog: no global injector              | no package-level service state in `internal/reviewd` (only error sentinels + one regex)                | ✅           |
| DO-6 analog: self-contained shutdown         | `Store.Close` touches only its own backend; no cross-service shutdown                                  | ✅           |
| DO-5 analog: no per-iteration construction   | `Pass` resolves nothing per view; services are built once per command                                  | ✅           |
| DO-1 analog: no panicking resolve at runtime | all construction propagates errors through `openConfiguredPipeline` into exit codes                    | ✅           |

**Should the repo adopt samber/do v2? No (YAGNI).** The dependency graph is 4 services with a single composition root and no lifetime complexity (no request/tenant scopes, no override-in-test needs — tests substitute via interfaces and constructor injection). A container would add an import and a second mental model without removing a single line of wiring pain. Revisit only if the daemon grows: an HTTP admin surface, multiple named reviewers, or per-request agent scopes.

## 3. Health-check architecture, before this pass

Three layers existed:

1. **`visionreviewd doctor`** (commands.go:436) — 5 probe families, each `name/ok/detail`, exit 1 on any failure: dataDir writable, reviewsDir writable, per-project glob match counts, journal read-and-fold (lock-aware: a running daemon's lock is skipped, a corrupt journal is failed with its event position), model endpoint `/models` + configured-model membership. Well built: injected writers, timeout bounds, covered by `main_test.go`.
2. **systemd readiness + restart** (nixos/visionreviewd.nix) — llama-server unit gates on `/health` (503 until weights loaded, `--retry 720`), daemon unit `Restart=on-failure` + `RestartSec=30s`, both `MemoryMax`-capped, full hardening.
3. **Event-sourced self-repair** — `Replay` rebuilds markdown byte-identically from the journal; `Backup` gives consistent snapshots; golden-journal pins guard the wire format.

**The gap:** all health checking was _pull-based and external_. Inside the daemon loop, a dead llama-server meant every pass ran scan → ingest → **one failed model call per changed view**, per-view errors joined and logged, forever. Resilient (no crash, no data loss) but noisy and wasteful — the daemon never _noticed_ the endpoint was down.

## 4. What this pass changed (implemented + tested)

| Change                                                                                                                                                                                           | File(s)                                       |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------- |
| `ProbeModelEndpoint(ctx, config, timeout) error` — the doctor's endpoint check extracted into the daemon package, sentinel `ErrEndpointUnhealthy` wrapped into every failure, messages preserved | `internal/reviewd/health.go`                  |
| `HealthGate` — `PassRunner` decorator: probe → delegate, or skip the pass with one warning; dead context bypasses the probe so shutdown keeps its accurate "skipped, pass context done" errors   | `internal/reviewd/health.go`                  |
| Doctor now delegates to the shared probe (single source of endpoint truth; output byte-identical)                                                                                                | `cmd/visionreviewd/commands.go`               |
| Daemon loop wired through the gate; `once`/`compare`/`replay` stay ungated (operator commands want full per-view diagnostics)                                                                    | `cmd/visionreviewd/commands.go` `gatedRunner` |
| Tests: probe table (listed / not-listed / 5xx / malformed / unreachable / API-key header), gate skip/delegate/dead-context/wiring sentinels                                                      | `internal/reviewd/health_test.go`             |
| Domain terms **Endpoint probe**, **Health gate**                                                                                                                                                 | `docs/DOMAIN_LANGUAGE.md`                     |

Design notes:

- The gate is a decorator over the **existing `PassRunner` seam** — `Daemon` and `Pipeline` are untouched; the seam existed for exactly this ("future consumers can substitute").
- `EndpointProbeTimeout` (5s) is shorter than doctor's 10s: a skip decision must fail fast; a diagnosis may wait.
- Gate errors wrap `ErrEndpointUnhealthy` themselves, so `errors.Is` holds for any probe implementation, not just the built-in one.
- Deferral is safe by construction: captures are content-addressed and deduped on hash, so views a skipped pass missed are picked up on the next healthy pass. No data loss.

## 5. SOA / composable / resilient / self-healing — scored

| Claim            | Evidence for                                                                                                                                                                                                    | Evidence against / notes                                                                                                                      |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| Service-oriented | ✅ clean service boundaries as Go packages + interfaces (`PassRunner`, `Reviewer`, `Store`, `BlobStore`, `Writer`)                                                                                              | It is a **modular monolith**, not network SOA — the right size for the job; systemd composes the _deployment_ units                           |
| Composable       | ✅ decorator pattern now proven twice (`PassRunner` gate; pipeline options); interfaces consumed by black-box tests                                                                                             | `Pipeline` fields are concrete types (fine at this size; an interface would be ceremony)                                                      |
| Resilient        | ✅ per-view error isolation (`errors.Join`), ctx checks per project+view, auto-compare failure never blocks review, non-overlapping passes, graceful SIGINT/SIGTERM, classified model errors with `IsRetryable` | `Config.Retry` is not wired into the daemon's reviewer (retry policy lives SDK-side)                                                          |
| Self-healing     | ✅ **now**: endpoint down → pass skipped with one warning → automatic recovery next interval; ✅ journal replay + backup + doctor; ✅ systemd restart                                                           | **KNOWN RED** `TestApplyViewStateIgnoresUnknownEventType` needs a product decision (tolerate-vs-strict fold), documented since pre-2026-10-05 |

## 6. Incident found and fixed during this review: SEVENTH jsonv2 recurrence

The review's lint step caught a **live seventh recurrence** of the known auto-commit jsonv2 migration bug (AGENTS.md §SDK):

- Offender: `fad59ce` (2026-10-11, auto-commit "Daemon: true") — re-migrated 7 files to `encoding/json/v2` / `encoding/json/jsontext`, including wire-format files (`pkg/vision/a2ui/component.go`, `messages.go` with the `omitzero` tag change and `jsontext.Value` swaps) — **three weeks undetected on master**.
- Also found: the 2026-10-05 restoration (`febc656`) had **missed `internal/reviewd/fakeserver_test.go`** — it carried v2 imports since `6390b14` (2026-09-29) despite the "tree-wide grep clean" claim.
- Fix: all 7 files restored to the v1 path (`encoding/json` only; `json.MarshalIndent`; `json.RawMessage`; `omitempty`). Committed by the auto-commit daemon as `e976164`; tree-wide grep = **0 v2 imports**. Verified: 6 of 7 files byte-identical to the blessed `febc656` state, `fakeserver_test.go` identical to its pre-recurrence ancestor except a whitespace alignment.
- Takeaway for AGENTS.md: the recurrence detector must be **scheduled**, not memory-based — see recommendation R1.

Bonus fixes in the same sweep:

- `pkg/vision/errors.go` re-exported only 14 of 16 `ErrorKind` kinds — `KindOverloaded`/`KindPaymentRequired` were missing, which made the CLI's `adviceForKind` switch unable to handle them (the `exhaustive` lint finding). Re-exports completed; advice strings added for all three missing kinds (`cmd/vision/main.go`).

## 7. Verification (executed this pass)

| Check                                          | Result                                                                                                                                                                                                                                                                   |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `go build ./...`, `go vet ./...`, `gofmt -l .` | clean (run post-fixes)                                                                                                                                                                                                                                                   |
| `go test -race ./...`                          | 8/9 packages green; the only failure is the documented KNOWN RED (`TestApplyViewStateIgnoresUnknownEventType`, untouched per policy)                                                                                                                                     |
| `nix run .#lint` (pinned golangci-lint)        | **1 issue** — the `err113` on `store.go:97`, which is the exact line of the KNOWN RED (product decision pending). Started this pass at 11 issues                                                                                                                         |
| `go mod tidy -diff` / `go mod verify`          | clean / all modules verified                                                                                                                                                                                                                                             |
| New tests                                      | `TestProbeModelEndpoint` (+subtests), `TestProbeModelEndpointUnreachable`, `TestProbeModelEndpointSendsAPIKey`, `TestHealthGate{SkipsPassWhenUnhealthy,DelegatesWhenHealthy,DeadContextBypassesProbe}`, `TestNewHealthGateRejectsInvalidWiring` — all pass under `-race` |

## 8. Recommendations (ranked)

| #  | Recommendation                                                                                                                                                                                                                                                                                                                      | Effort |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| R1 | The CI grep guard exists (first step of `build-and-test`) but **auto-commit direct pushes bypass PR checks by design** — that is exactly how seven recurrences landed. Make the auto-commit daemon itself run the tree-wide grep + pinned lint before committing (or add a scheduled job so drift is caught within days, not weeks) | M      |
| R2 | Decide the KNOWN RED fold semantics (tolerate-and-ignore vs strict) — it is the only red test and the only `err113` finding                                                                                                                                                                                                         | M      |
| R3 | Consider gating `once` with the same health gate behind a flag (`--skip-unhealthy`) if operators start running it from cron                                                                                                                                                                                                         | S      |
| R4 | If the daemon ever grows an HTTP surface, reuse `ProbeModelEndpoint` + the doctor checks as a `/healthz` handler — the probe extraction made that a one-liner                                                                                                                                                                       | S      |
| R5 | Do NOT adopt samber/do v2 now; revisit at ≥ 2 composition roots or per-request lifetimes (see §2)                                                                                                                                                                                                                                   | —      |

## 9. Answer, in one sentence each

- **samber/do v2 superbly?** Not used — and the manual constructor injection it would replace is already exemplary (validated sentinels, interfaces at every seam, complete shutdown discipline).
- **Proper SOA?** A properly-bounded modular monolith: services are packages, seams are interfaces, systemd is the composition layer.
- **Composable?** Yes — proven concretely today by adding a `PassRunner` decorator with zero changes to `Daemon` or `Pipeline`.
- **Resilient?** Yes — per-view isolation, context discipline, classified errors, non-overlapping passes.
- **Self-healing?** Now genuinely yes at the model-endpoint boundary (health gate), and at the data layer (replay/backup/doctor); the remaining self-healing gap is the undecided unknown-event fold semantics (R2).
