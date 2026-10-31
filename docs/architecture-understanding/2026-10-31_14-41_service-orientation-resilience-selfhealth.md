# Service Orientation, Composability, Resilience & Self-Health — Deep Review

**Date:** 2026-10-31 · **Scope:** `internal/reviewd`, `cmd/visionreviewd`, `pkg/vision` (daemon-facing surface), `nixos/visionreviewd.nix`
**Trigger question:** *"Are we using samber/do v2 in combination with Health Checks superbly?! Do we have a PROPER Service Oriented, composable, resilient, and self-health architecture?"*
**Method:** architecture-review skill rubric (7 dimensions, 1–5) + samber-do-best-practices skill (DO-1 → DO-6) applied to the actual composition style.

---

## 1. Executive Summary

| Question | Verdict |
| --- | --- |
| **Are we using samber/do v2 superbly?** | **We are not using samber/do at all.** Zero imports, zero go.mod entries (verified by grep + `go.mod`). The premise is false — and for this codebase that is *mostly the right call* (§2). |
| **Proper service-oriented, composable, resilient, self-health architecture?** | **Good, not superb — overall 3.9/5.** Event sourcing as truth, per-view error isolation, atomic writes everywhere, and (as of today) a runtime HealthGate put this well above typical daemon quality. What keeps it from "superb": no retry policy wired into the daemon, no liveness/watchdog signal, health is skip-not-signal, and only the top seam (`PassRunner`) is interface-abstract. |

**One-sentence answer:** This is a cleanly composed, idiomatically-wired, resilient single-process daemon whose self-health story matured *during this review* (a runtime health gate landed mid-audit), but which still lacks in-process retry wiring and any externally observable liveness signal — and it does all of this *without* a DI container, which is defensible here.

---

## 2. samber/do: Absence Analysis (Q1)

Facts:

- `rg 'samber/do' --include='*.go'` → **0 hits**; `go.mod` → no `github.com/samber/do` direct or indirect.

What the daemon uses instead — **plain constructor injection with sentinel-validated wiring**, composition root in `cmd/visionreviewd/commands.go:58` (`openPipeline`):

```text
Config (JSON, validated+normalized)
  └─ NewReviewerFromConfig ── fantasy.LanguageModel (openaicompat)     reviewer.go:48
  └─ OpenStore ─────────────── go-cqrs-lite bbolt backend              store.go:191
  └─ NewBlobStore ─────────── content-addressed blob store             blobstore.go:20
  └─ NewWriter ────────────── atomic markdown writer                   writer.go:28
       └─ NewPipeline(reviewer, store, blobs, writer, logger)          pipeline.go:59
            └─ NewHealthGate(pipeline, probe, logger)  [run only]      commands.go:204
                 └─ NewDaemon(runner, projects, interval, logger)      daemon.go:35
```

Applying the DO-1 → DO-6 anti-pattern rubric to this style (spirit, not library):

| Rule | Status | Evidence |
| --- | --- | --- |
| DO-1 `Must*` on runtime paths | ✅ pass | No `Must*` anywhere; every constructor returns `error` (`NewDaemon`, `NewPipeline`, `NewHealthGate` all reject nil wiring with sentinels: `daemon.go:13-17`, `pipeline.go:14-19`, `health.go:17-21`) |
| DO-2 `New` without `Shutdown` | ✅ pass (manual) | `Store.Close()` (`store.go:224`); every CLI command `defer closeStore(...)` (`commands.go:97,213,419`); `openPipeline` closes the store on partial-failure (`commands.go:77`) |
| DO-3 `Override*` outside tests | ✅ n/a | No container, no overrides; test seams are constructor parameters (fake `fantasy.LanguageModel`, `gateRecorder`, counting `PassRunner`) |
| DO-4 package-level injector/global | ✅ pass | No globals holding wiring; composition happens per-command in `commands.go` |
| DO-5 `Invoke` in loops | ✅ n/a | Dependencies resolved once at construction, held as fields |
| DO-6 `Shutdown` touching other services | ✅ pass | `Store.Close` is self-contained; no cross-service shutdown ordering exists (only one closable service) |
| Lifecycle guards (`var _ do.Shutdowner…`) | ⚠️ missing | No compile-time conformance checks; lifecycle is by-convention (`Close()`), not by-interface |
| `do.Healthchecker` per service | ⚠️ missing | No per-service `Health()` methods; health is command-level (doctor) + decorator-level (HealthGate), not a registry |

**Judgment — do NOT adopt samber/do here, today.** The daemon has exactly **one** resource-holding service (the bbolt `Store`); a container's value (ordered shutdown, health registry, scopes, named services) pays off with *many* lifecycle-bearing services. Introducing one now would add a dependency and indirection for zero behavioral gain. Revisit **only if** the service graph grows (e.g., multiple stores, HTTP surface, worker pools). The DO-1..DO-6 spirit is already satisfied better than most container-using codebases.

---

## 3. What the Architecture Actually Is (inventory)

**Shape:** single binary, 9 subcommands (`run once discover compare events backup replay doctor version`, `main.go:35-46`), one daemon loop, event-sourced state, markdown as a *projection*.

**Data ownership (exemplary):**

- **Journal (bbolt) is the single source of truth** — every capture/review/comparison is an event on stream `<project>:<viewKey>`; `Replay` rebuilds the entire reviews dir byte-identically (`replay.go:76-80`, pinned by golden-journal tests).
- **Blob store** is content-addressed (`<sha256>.<ext>`, `blobstore.go:39-64`) — BEFORE images survive in-place screenshot overwrites.
- **Markdown** is disposable projection; INDEX timestamps derive from folded state, never `time.Now()` — deterministic replay.

**Lifecycle map:**

| Unit | Resource | Lifecycle | Owner |
| --- | --- | --- | --- |
| `Store` | bbolt file handle | `Close()` | per-command `defer` |
| `Reviewer` | 2 vision agents (stateless clients) | none needed | — |
| `BlobStore`/`Writer` | filesystem only | atomic per-op (temp+rename) | — |
| `Daemon` | goroutine + ticker | ctx cancel; `Run` returns nil on SIGINT/SIGTERM | `runDaemonCommand` |
| `llama-server` (external) | model weights | systemd unit + `/health` readiness gate + `Restart=on-failure` | NixOS module |

**Seams (composability surface):**

- `PassRunner` interface (`daemon.go:21`) — consumed by Daemon; implemented by `*Pipeline` **and** `*HealthGate` (decorator) and test stubs. This is the one true abstraction seam and it proved itself the day it was needed.
- `fantasy.LanguageModel` — provider behind an interface (`reviewer.go:59`); fakes drive E2E tests (`fakeserver_test.go`).
- `vision.Analyzer` (SDK) — mockable analysis surface.
- Concrete everywhere else: `Pipeline` holds `*Reviewer`, `*Store`, `*BlobStore`, `*Writer` (`pipeline.go:45-55`); `ScanProject` is a package function with filesystem coupling (`scan.go:34`).

---

## 4. Scored Assessment (rubric, evidence-cited)

| Dimension | Score | Justification |
| --- | --- | --- |
| **Coupling** | **4** | Key seams interface-driven (`PassRunner`, `LanguageModel`, `Analyzer`); no import cycles; domain fold (`ApplyViewState`) is pure. Held back: Pipeline→concrete types, `ScanProject` un-injectable, doctor re-derives config context inline. |
| **Cohesion** | **5** | One concept per file (viewkey/hash/blobstore/scan/score/prompts/markdown/writer/health); package `reviewed` is a single bounded context; SDK (`pkg/vision`) cleanly separated from daemon (`internal/reviewd`). |
| **Modularity** | **4** | `pkg/` (public SDK) vs `internal/` (daemon, catalog) vs `cmd/` boundaries enforced by Go visibility; a2ui is a self-contained sub-package with machine-pinned conformance schemas. Minor: `internal/reviewd` is one flat 14-collaborator package — right-sized today. |
| **Composability** | **4** | Decorator pattern proven in production code (`HealthGate` wraps `Pipeline` wraps nothing-but-could); config externalized; per-persona agents composed from one model. Gap: mid-layer deps are concrete, so new runners compose only at the top seam. |
| **Service Orientation** | **4** | Bounded context owns its data end-to-end (journal+blobs+projection); each subcommand is a service with explicit contract (exit codes 0/1/2, `main.go:26-32`); the external model service has an explicit API contract (OpenAI-compat `/v1`) with dependency ordering (`after=`/`wants=` + readiness gate, `visionreviewd.nix:79-80,131-137`). It is a monolith — but one with extraction-ready internals. |
| **Dependency Direction** | **5** | Domain (events, `ViewState`, prompts, score parsing) imports nothing infra; store behind go-cqrs-lite; provider behind fantasy interface; `internal/catalog` isolates catwalk; CLI→internal only, never reverse. Third-party deps isolated behind interfaces at every boundary. |
| **Scalability** | **3** | Sequential per-view processing; one bbolt writer (lock-enforced single daemon); model calls dominate and a hung view blocks the pass for up to `Timeout` (5 min default) each. Perf baseline shows scan+fold is free (~2.9ms/100 views) — the wall is serial inference. SDK already ships `AnalyzeBatch` (semaphore-bounded) that the daemon doesn't use. |
| **Resilience** (central) | **4** | See §5. |
| **Self-health** (central) | **3.5** | See §6. |

**Overall: 4.0 → "Good"** (rubric band 3.5–4.4: address lowest-scoring dimension; no urgent restructuring). Excluding the two bonus dimensions: 4.0 exactly.

---

## 5. Resilience Audit (the strongest part of this codebase)

### Present and verified

1. **Per-view error isolation** — `Pipeline.Pass`/`passProject` collect per-view failures via `errors.Join` and continue (`pipeline.go:114-145,158-211`); one broken view or model failure never blocks others; INDEX still refreshes (renders `?` for failed rows).
2. **No permanent skip after transient outage** — skip-seen keys on "reviewed-current-hash" (`NeedsReview`, `pipeline.go:174-182` + `store.go:46-48`), so a view whose review failed once is retried next pass. This is event-sourced *eventual* retry with a 10-minute cadence.
3. **Fail-fast cancellation** — dead context produces explicit "skipped, pass context done" errors wrapping `ctx.Err()` (`pipeline.go:117-121,161-165`); INDEX refresh skipped under dead context to avoid noise; `Daemon.Run` distinguishes graceful cancel from unexpected ctx death (`daemon.go:77-83`).
4. **Atomicity everywhere** — blob writes temp+rename (`blobstore.go:68-111`), markdown writes temp+rename+chmod (`writer.go:79-113`); readers never see partial files; permissions tightened per file (0o600/0o640).
5. **Rebuildability** — `Replay` regenerates the reviews dir byte-identically; `backup` takes a consistent bbolt snapshot; `doctor` folds the whole journal to prove integrity; error paths name journal path + event position.
6. **Runtime endpoint gate (landed today)** — `HealthGate` skips a whole pass when `/v1/models` doesn't list the configured model, instead of failing every changed view against a dead endpoint; self-heals next interval (`health.go:88-146`, wired in `commands.go:204-221`). Dead-context bypass keeps shutdown semantics honest (`health.go:124-132`, pinned by `TestHealthGateDeadContextBypassesProbe`).
7. **OS-level** — `Restart=on-failure` + `RestartSec` on both units; llama-server has a `/health` readiness gate with ~1h retry window so the daemon cannot race the first-start model download (`visionreviewd.nix:131-140`); sane sandboxing + `MemoryMax` on both.
8. **Classified errors (SDK)** — `*ModelError` with 16 `ErrorKind`s + `IsRetryable()` (`pkg/errors/model.go`), so *consumers can* implement retry policy — the building block exists.

### Missing (ranked)

1. **No retry policy is wired into the daemon.** `vision.Config.Retry` (backoff+jitter, retryable-aware) exists in the SDK, but `NewReviewerFromConfig` → `newAgent` (`reviewer.go:75-91`) never sets it, and `MaxRetries` defaults to 0 (no HTTP-layer retry). One transient 500 = a failed view; the earliest retry is the *next pass* (default 10 minutes away). For a daemon whose dependency is a local llama-server that restarts, this is the single highest-value resilience gap.
2. **No liveness/watchdog signal.** `Type=simple` (default) + no `WatchdogSec`/`sd_notify`: a hung model call (up to `Timeout` per view, serially) leaves the daemon "active" and systemd blind. The HealthGate probe bounds the *endpoint-dead* case at 5s, not the *request-hung* case.
3. **No pass-overlap guard beyond the loop shape** — passes are serial by construction (good), but a pass longer than `interval` silently skips ticks (ticker drops); no warning is logged when that happens.
4. **Serial inference caps throughput** — see Scalability 3; fine for 17 sites, structural at 100+.

---

## 6. Self-Health Audit (immature — but improving in real time)

The health architecture is **three layers**, only two of which existed before today:

| Layer | Mechanism | Mode | Gap |
| --- | --- | --- | --- |
| **Offline diagnosis** | `visionreviewd doctor` — writable dirs, per-project glob counts, journal deep-read+fold, model-endpoint probe (`commands.go:534-543`) | manual, CLI | on-demand only |
| **Runtime gate** | `HealthGate` per-pass probe (5s budget) + skip-with-warning (`health.go`, `commands.go:204`) | automatic, per pass | skip-not-signal: no external observer learns the daemon is degraded |
| **Process supervision** | systemd `Restart=on-failure` both units; llama readiness gate | automatic | restart covers *exit*, not *hang* (no watchdog) |
| **Live signal** | — | — | **absent**: no `/healthz`, no sd_notify `READY=1`/`WATCHDOG=1`, no metrics, no pass-outcome counters; pass results exist only as slog lines |

Notably, doctor and the runtime gate now **share one probe** (`ProbeModelEndpoint`, `health.go:32`) — a dedup the daemon refactored correctly (previously a 70-line inline copy in `checkModelEndpoint`).

**Self-monitoring blind spot:** a view that stays unreviewed for N passes (persistent model rejection, e.g. content filter on one screenshot) is invisible without grepping logs — the journal *has* the data (`NeedsReview` true), nothing aggregates it.

---

## 7. Incidents Found *During* This Review (process findings)

1. **7th `encoding/json/v2` migration recurrence — restored & verified.** Auto-commit `fad59ce` (2026-10-11) re-migrated 7 files to `encoding/json/v2`/`jsontext` (incl. the SDK-subset `pkg/vision/a2ui`), six days after the documented 2026-10-05 restore. This review restored all 7 to the v1 path (`jsontext.Value`→`json.RawMessage`, `omitzero`→`omitempty`, `MarshalWrite/UnmarshalRead`→Encoder/Decoder, `jsontext.WithIndent*`→`MarshalIndent`). Verified: tree-wide grep clean, `go build ./...` + `go vet ./...` green, tests green (`internal/reviewd`, `pkg/vision/a2ui`, `cmd/*`), race green on touched pkgs, `go mod tidy -diff` clean, **depguard green** under golangci-lint.
2. **Concurrent auto-commit daemon interference.** While this review ran, the daemon *authored* new features in-commit: `health.go` + `health_test.go` (health gate), the `gatedRunner` wiring, doctor probe dedup, and CLI advice additions (`6b26646`, `fad04d8`, `f0cb749`, `426569b`, `f4bb7b8` — all 2026-10-31, 14:21–14:36). A full-suite run caught a **mid-race build failure** (`undefined: vision.KindOverloaded`) that was gone 90 seconds later — the daemon had written `cmd/vision/main.go` before its surrounding state settled. The work itself is good (assessed in §5–6 as found); the process risk is that these commits bypass required PR checks by design, and today one of them broke the build for the duration of a file write.
3. **Lint version drift (known class).** Local binary is golangci-lint **2.14.0**; CI pins **2.13.2**. Today's local run shows 2 findings (`err113` on `store.go:97` — documented known-red companion; `gochecknoglobals` on `adviceByKind`, introduced by today's auto-commit `f4bb7b8`). Per repo policy, CI and local must move together — 2.14.0 is unpinned locally.
4. **Known red test remains** — `TestApplyViewStateIgnoresUnknownEventType` (fold-strictness vs tolerate-and-ignore; product decision, documented pre-2026-10-05). Still the only test failure on the tree.

---

## 8. Action Roadmap

| Prio | Item | Where | Why / acceptance |
| --- | --- | --- | --- |
| **P0** | **Wire a retry policy into the daemon's agents** — set `vision.Config.Retry` (e.g. 2 attempts, 30s→2m backoff+jitter) in `newAgent`/`NewReviewerFromConfig`, configurable via daemon `Config` | `internal/reviewd/reviewer.go:75-91`, `config.go` | Closes the top resilience gap; SDK already honors `IsRetryable` and skips retry on streaming (n/a here). Accept: fake-server test asserts a transient 500 is retried within one pass, not deferred to the next interval. |
| **P0** | **Decide the fold-strictness question** — tolerate-and-ignore (match the test, forward-compatible journals) vs strict fold (match the code, delete the test) | `store.go:97` + `store_test.go:333` | Unblocks the only red test; either answer also resolves the repo's only `err113` finding. |
| **P1** | **Add a liveness signal** — `Type=notify` + `sd_notify(WATCHDOG=1)` heartbeat after each pass (or per-view), `WatchdogSec` > worst pass time; NixOS module update | `cmd/visionreviewd`, `nixos/visionreviewd.nix` | Converts hang-detection from "never" to "bounded"; pairs with HealthGate (endpoint-dead) which already bounds that class at 5s. |
| **P1** | **Pass-duration observability** — log/warn when a pass exceeds `interval` (dropped ticks) and expose per-pass counters (captured/reviewed/failed) in one summary line | `internal/reviewd/daemon.go` `runPass` | Cheap; makes degradation greppable and later wireable to a health endpoint. |
| **P2** | **Unreviewed-lag monitor** — fold `NeedsReview` counts into the pass summary ("2 views pending >3 passes") from journal data already loaded | `pipeline.go` / new query on `Store` | Turns the §6 blind spot into a log-visible (later alert-able) signal without new state. |
| **P2** | **Consider parallel per-view reviews** with the SDK's semaphore-bounded pattern (`AnalyzeBatch`) once fleet size makes serial passes exceed interval | `pipeline.go:passProject` | Only after P0: retry-first, then concurrency; keep per-view isolation semantics. |
| **P3** | **Interface-ify mid-layer deps** (`ViewRecorder`, `Blobs`, `ReviewWriter`) only if a second implementation or in-memory test double is actually needed | `pipeline.go:45-55` | Current real-FS/bbolt tests are a feature, not a bug; don't abstract speculatively. |
| **P3** | **Revisit samber/do** only when ≥3 lifecycle-bearing services exist | — | Per §2: adding it now is cost without benefit. |

---

## 9. Verification Evidence (this review's own runs)

- `rg -l 'encoding/json/(v2|jsontext)'` → **clean** (post-restore)
- `go build ./...`, `go vet ./...` (go 1.27.1, nix-store) → **ok**
- `go test ./internal/reviewd/ ./pkg/vision/a2ui/ ./cmd/visionreviewd/ ./cmd/vision/ ./pkg/...` → **all ok** except documented known-red `TestApplyViewStateIgnoresUnknownEventType`
- `go test -race ./pkg/vision/a2ui/ ./cmd/visionreviewd/` → **ok**
- `go mod tidy -diff` → **empty**
- `golangci-lint run` (local 2.14.0) on touched packages → **2 pre-existing findings**, depguard **green**
- Mid-review build race (daemon concurrent write) reproduced once at 14:36, gone on re-run — see §7.2

---

*Point-in-time snapshot. The auto-commit daemon mutates this tree continuously; re-verify §9 commands before citing scores elsewhere. Roadmap items harvested to `TODO_LIST.md`. The health-gate implementation record (from the concurrent daemon pass that authored `health.go`) lives at `docs/status/2026-10-31_14-45_samber-do-health-architecture-review.md` — same verdict on samber/do, complementary detail on the fix.*
