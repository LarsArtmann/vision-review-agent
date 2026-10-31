# Session Status: samber/do + Health-Check Architecture Review, HealthGate, Seventh jsonv2 Restoration

**Date:** 2026-10-31 17:27 CET · **Session window:** ~14:00–14:50 UTC+1 · **Scope of this report:** this session's work only (per instruction — no new research)

**Trigger:** "Are we using samber/do v2 + Health Checks superbly? SOA/composable/resilient/self-healing?" — deep review requested, report delivered, fixes executed and verified.

---

## Executive summary

1. **The premise failed honestly:** samber/do v2 is not used anywhere (only `samber/lo`, transitively). The daemon's manual constructor injection passes every DO-1→DO-6 analog. Adoption verdict: YAGNI.
2. **One real architecture gap existed and is now fixed:** health checking was pull-only (doctor CLI + systemd). A dead model endpoint meant one failed model call per changed view, every pass, forever. `HealthGate` (PassRunner decorator) + `ProbeModelEndpoint` now skip-and-self-heal. Implemented, unit-tested, verified.
3. **The lint step surfaced a live incident:** the SEVENTH jsonv2 recurrence (`fad59ce`, 2026-10-11, auto-commit) had re-migrated 7 files incl. a2ui wire code — undetected for 3 weeks. Plus proof the 2026-10-05 restoration's "tree-wide grep clean" claim was **false** (`fakeserver_test.go` carried v2 since 09-29). All restored to the v1 path this session.
4. **Bonus catch:** `pkg/vision` re-exported only 14/16 `ErrorKind`s — `KindOverloaded`/`KindPaymentRequired` were unreachable from the CLI's advice path. Completed.
5. **Final verified state:** build/vet/gofmt clean · race suite 8/9 packages green (only the documented KNOWN RED) · pinned lint **11 → 1 issues** (the 1 is the KNOWN RED's own line) · `go mod tidy -diff` + `go mod verify` clean.

---

## a) FULLY DONE (verifiable, committed)

| # | What | Evidence | Files |
|---|------|----------|-------|
| 1 | samber/do absence proven; DO-1→DO-6 audit of manual-DI analogs — all pass | `go.mod`/`go.sum`/`go mod graph` checked; audit table in report | — |
| 2 | `ProbeModelEndpoint` extracted from the doctor CLI into `internal/reviewd` — single source of endpoint truth, sentinel `ErrEndpointUnhealthy` wrapped into every failure, messages preserved | builds + doctor output byte-identical (main_test.go green) | `internal/reviewd/health.go`, `cmd/visionreviewd/commands.go` |
| 3 | `HealthGate` PassRunner decorator wired into `run` only; dead-context bypass preserves shutdown semantics; `once`/`compare`/`replay` deliberately ungated | `errors.Is(err, ErrEndpointUnhealthy)` contract tested | `internal/reviewd/health.go`, `cmd/visionreviewd/commands.go` |
| 4 | 7 new tests, all green under `-race`: probe table (listed/not-listed/5xx/malformed/unreachable/API-key), gate skip/delegate/dead-context/nil-wiring | `go test -race ./internal/reviewd/` | `internal/reviewd/health_test.go` |
| 5 | Seventh jsonv2 recurrence restored: 6 files byte-identical to blessed `febc656` state via `git restore --source=febc656`; `fakeserver_test.go` import block hand-fixed to v1 | tree-wide grep = **0** v2 imports; `pkg/vision/a2ui` + `internal/reviewd` race suites green; committed as `e976164` | 7 files (config*, discover, events_test, fakeserver_test, a2ui component/messages) |
| 6 | `ErrorKind` re-exports completed 14/16 → 16/16 (`KindOverloaded`, `KindPaymentRequired`); CLI advice table for all 16 kinds; switch → map (cleared `exhaustive` + `cyclop`) | `cmd/vision` tests green; lint finding gone | `pkg/vision/errors.go`, `cmd/vision/main.go` |
| 7 | wrapcheck fix: `HealthGate.Pass` wraps delegated inner errors (`gated pass: %w`) | lint finding gone | `internal/reviewd/health.go` |
| 8 | Architecture review report with DO-audit, health inventory, gap analysis, verdicts | committed | `docs/status/2026-10-31_14-45_samber-do-health-architecture-review.md` |
| 9 | Domain vocabulary: **Endpoint probe**, **Health gate** | committed | `docs/DOMAIN_LANGUAGE.md` |
| 10 | AGENTS.md: health-gate bullet, SEVENTH-recurrence paragraph incl. the false-2026-10-05-claim correction, ErrorKind re-export bullet | committed | `AGENTS.md` |
| 11 | Verification matrix steps 1–6 executed post-fixes: build/vet/gofmt clean, race 8/9, pinned lint **1 issue** (the KNOWN RED's err113 line), tidy-diff clean, mod verify all-ok | job outputs this session | — |

## b) PARTIALLY DONE

| What works | What remains | Blocker | Effort |
|---|---|---|---|
| Health gate at unit level (probe + decorator + wiring) | **No E2E test through `visionreviewd run`** with a server that dies mid-session and recovers; gate timeout (5s) and on/off are hard-coded, not config.json fields | none — just not built yet | M / S |
| jsonv2 restoration | GOEXPERIMENT=none SDK-subset build **not re-run** this session to prove remaining failure is ONLY the transitive kaptinlin error; nix steps 7–9 of the verification matrix not re-run | KNOWN BLOCKER (fantasy ≥ v0.45.2 floors kaptinlin v0.9.9) for the former; time for the latter | S / M |
| Seventh-recurrence root cause | Offender commit identified (`fad59ce`) but two of its side effects **uninvestigated**: `go.sum` −92 lines, `SECURITY.md` +40 lines | none — explicitly out of scope per "no unrelated research" | S |
| Recommendation R1 (stop the recurrence class) | Written up (gate the auto-commit daemon / scheduled grep) but **not implemented** — 7 recurrences prove memory-based detection is dead | infra decision needed (see question g2) | M |
| LSP diagnostics hygiene | 4 standing LSP warnings (depguard/gci/gocognit) are CLI-verified stale, but LSP was never restarted to confirm they clear | none | S |
| HARVEST | This report's section (f) is the input for docs-health HARVEST into TODO_LIST.md/ROADMAP.md — **not done** | user said WAIT after the report | S |

## c) NOT STARTED (planned by this session, zero code)

- **R2:** product decision + fix for `TestApplyViewStateIgnoresUnknownEventType` (tolerate-and-ignore vs strict fold) — blocked on decision, see g1. It is the only red test AND the last lint finding.
- **R3:** `once --skip-unhealthy` flag for cron operators.
- **R4:** `/healthz` HTTP endpoint reusing the probe + doctor checks (only meaningful if the daemon gains an admin port).
- **R5:** samber/do v2 adoption — deliberately not started; revisit triggers documented (≥2 composition roots or per-request lifetimes).
- Circuit breaker / probe backoff after consecutive unhealthy passes.
- `visionreviewd doctor --json` for machine consumption (site-monitor integration).
- systemd watchdog (`WatchdogSec`/`sd_notify`) for a *hung* (not crashed) daemon.
- Retry wiring (`Config.Retry`) into the daemon's reviewer — flagged, not built.

## d) TOTALLY FUCKED UP (radical honesty)

1. **The jsonv2 "detection system" is a human-memory ritual and it is dead.** Seven recurrences; the seventh (`fad59ce`) sat on master for **3 weeks** inside wire-format files (a2ui `messages.go` `omitzero`/`jsontext.Value`). The CI grep guard cannot help because auto-commit direct pushes bypass PR checks **by design**. Nothing in the system — not lint-on-push (lint never runs on those commits), not CI (bypassed), not AGENTS.md (prose, unread by the daemon) — actually stands between the daemon and master. Severity: wire-format drift reachable in production. Mitigation so far: manual restoration + a paragraph in AGENTS.md. That paragraph will fail exactly like the last one.
2. **A false verification claim shipped into AGENTS.md.** "Tree-wide grep clean (verified 2026-10-05)" was **false** — `fakeserver_test.go` carried v2 imports from 09-29 through the "verification". A verification claim without a captured artifact (command output, timestamp) is trust-me-ware. This one cost 3 weeks of latent breakage.
3. **The KNOWN RED is now a 26-day-old open product decision blocking two gates.** `TestApplyViewStateIgnoresUnknownEventType` red since pre-2026-10-05; its `store.go:97` line is also the repo's ONLY remaining lint finding. Master carries a permanent red test + permanent lint noise because nobody decides tolerate-vs-strict.
4. **My own bugs this session (caught, but mine):** (a) `HealthGate.Pass` v1 did not classify skip errors — `errors.Is(err, ErrEndpointUnhealthy)` failed for custom probes until I wrapped the sentinel in the gate; (b) interface delegation without error wrapping (wrapcheck); (c) `adviceForKind` as a switch tripped `cyclop` after adding 3 kinds. All caught by tests/lint same-session, all fixed — but each was a design miss I should have avoided upfront.
5. **Unexplained anomaly, unresolved:** `fakeserver_test.go` showed v2 imports on disk, then v1, without an edit attributable to me — most likely the **parallel session** that landed `docs/architecture-understanding/2026-10-31_14-41_service-orientation-resilience-selfhealth.md` fixed it concurrently while sharing this working tree. Two agents + one auto-commit daemon on one checkout means interleaved commits and ambiguous authorship (commits `f4bb7b8`, `141a6b3` mix both sessions' work). Nothing broke — but the confusion cost me real minutes and I cannot fully reconstruct the sequence.

## e) WHAT WE SHOULD IMPROVE

1. **Make verification claims carry artifacts.** Every "verified X" in AGENTS.md should embed the command + output timestamp. Prose verification rots silently; this incident proves it.
2. **Gate the writer, not the reader.** Fixing recurrences after they land is Sisyphean; the auto-commit daemon should refuse to commit when the tree-wide grep or pinned lint fails. One gate at the commit boundary beats seven post-hoc restorations.
3. **Tests as guards, not docs.** A `_test.go` that greps its own package for `encoding/json/v2` runs on every `go test` everywhere, immune to CI bypass. Docs cannot do that.
4. **Trust the CLI, but also explain the LSP.** Four stale LSP warnings stood all session; I verified via CLI per policy but never restarted the LSP to close the loop. "Stale but present" warnings erode trust in every future warning.
5. **Design the error contract before the implementation.** Both of my initial mistakes (sentinel classification, delegation wrapping) were contract-shaped, not logic-shaped. Writing the `errors.Is` contract as a failing test first would have caught both pre-implementation.
6. **Skill fan-out worked — say so.** Loading `samber-do-best-practices` before acting gave the DO-audit vocabulary and canonical patterns for free. The failure mode it prevented (cargo-culting a DI container into a 4-service daemon) is worth keeping in the loop.
7. **Parallel sessions need a coordination norm.** A second agent on the same checkout + auto-commit daemon interleaved both sessions' commits. Before claiming file authorship or state, check `git log --author/time` first; better, note the expectation that sessions may overlap.

## f) 50 things to get done next (ranked; HARVEST input — route to TODO_LIST/ROADMAP)

*Impact: 🔴 Critical / 🟠 High / 🟡 Medium / ⚪ Low · Effort: S <30min / M 30min–2h / L >2h*

**Incident class: jsonv2 recurrence (highest leverage)**
| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Gate the auto-commit daemon: tree-wide v2-grep + pinned lint must pass before each auto-commit, else abort | 🔴 | M | Process |
| 2 | Add package-local guard tests that fail if `encoding/json/v2` appears in `internal/reviewd` and `pkg/vision` (runs on every `go test`, CI-bypass-immune) | 🟠 | S | Tests |
| 3 | Add scheduled weekly CI job running the v2-import grep + depguard over the whole tree | 🟠 | S | CI |
| 4 | Rewrite AGENTS.md "verified" claims to embed command + timestamp artifacts (start with the json bullet) | 🟡 | S | Docs |
| 5 | ANNOTATE the 2026-10-05 status doc: its "grep clean" claim was false for fakeserver_test.go (docs-health ANNOTATE, never rewrite) | 🟡 | S | Docs |
| 6 | Prove the GOEXPERIMENT=none SDK-subset failure contains ONLY the kaptinlin transitive error (our tree v1-clean under v1 regime) | 🟠 | S | Quality |
| 7 | Investigate `fad59ce`'s `go.sum` −92 lines — confirm nothing was silently dropped | 🟡 | S | Bug |
| 8 | Inspect the `SECURITY.md` (+40 lines) added by `fad59ce` — confirm intentional | ⚪ | S | Docs |
| 9 | Decide + implement `no-jsonv2` job scope: fantasy-free subset OR pin fantasy v0.41.1 (documented blocker, tracked) | 🟠 | M | CI |
| 10 | Track kaptinlin/fantasy upstream for a dual-regime release; bump when available | 🟡 | S | Deps |
| 11 | Document `omitempty` vs `omitzero` dual-regime wire semantics at the `sendDataModel` field in `messages.go` | 🟡 | S | Docs |

**Product decision (unblocks two gates)**
| 12 | Decide unknown-event fold semantics (tolerate vs strict) and fix `ApplyViewState` or the test — clears the only red test AND the last lint finding | 🔴 | S–M | Bug |

**Health gate maturation**
| 13 | E2E test: `run` against a fake server that dies mid-session and recovers; assert one warning, later recovery, no data loss | 🟠 | M | Tests |
| 14 | Make gate probe timeout + gate on/off configurable via config.json | 🟡 | S | Feature |
| 15 | Add gate observability: consecutive-skip counter, last-healthy timestamp in pass logs | 🟡 | S | Feature |
| 16 | Log healthy→unhealthy→healthy transitions at Info (state machine in logs) | ⚪ | S | Feature |
| 17 | `once --skip-unhealthy` flag for cron operators | ⚪ | S | Feature |
| 18 | Probe backoff: after N consecutive unhealthy passes, skip work entirely between probes (avoid 5s-probe spam) | ⚪ | S | Feature |
| 19 | Log WHICH projects/views a skipped pass deferred (next-pass pickup list) | ⚪ | S | Quality |
| 20 | Wire `Config.Retry` into the daemon's reviewer (vision-level backoff for transient model errors) | 🟠 | M | Feature |
| 21 | Reuse one `http.Client` for probes instead of per-call construction | ⚪ | S | Quality |
| 22 | Refresh INDEX.md semantics under a skipped pass — document (or fix) that INDEX is not refreshed | 🟡 | S | Quality |
| 23 | `/healthz` endpoint reusing probe + doctor checks (when the daemon gains an admin port) | 🟡 | M | Feature |
| 24 | `visionreviewd doctor --json` for machine consumption (site-monitor) | 🟡 | S | Feature |
| 25 | Compile-time `var _ PassRunner = (*HealthGate)(nil)` assertion (house pattern) | ⚪ | S | Quality |
| 26 | Early disk-health probe in `Pass` (skip pass when blob/reviews dir unwritable instead of per-view write errors) | 🟡 | S | Feature |
| 27 | systemd watchdog (`WatchdogSec` + `sd_notify`) so a HUNG daemon also restarts | 🟡 | M | Feature |
| 28 | Per-glob error isolation in `ScanProject` (one bad pattern shouldn't fail a project's whole pass) | ⚪ | S | Quality |
| 29 | Wire site-monitor.sh to alert on consecutive gate-skip warnings in daemon logs | 🟡 | M | Feature |
| 30 | Re-run perf baseline incl. gate probe cost per pass; refresh the M14 doc numbers | ⚪ | S | Quality |

**DI / samber-do follow-through**
| 31 | Record the "no DI container by design" decision in `docs/decisions/` so the question doesn't recur | 🟡 | S | Docs |
| 32 | Revisit samber/do only on trigger: ≥2 composition roots or per-request lifetimes (verbatim from R5) | ⚪ | — | Feature |

**Docs + process debt**
| 33 | HARVEST this report's section (f) into TODO_LIST.md / ROADMAP.md (docs-health HARVEST) | 🟠 | S | Docs |
| 34 | Update FEATURES.md with the health gate + probe | 🟡 | S | Docs |
| 35 | Add a short "seventh recurrence" post-mortem note to docs/DEPS.md or a lessons file | 🟡 | S | Docs |
| 36 | Restart LSP and confirm the 4 stale warnings clear (close the CLI-vs-LSP loop) | ⚪ | S | Quality |
| 37 | Doctor failure-detail assertions: extend main_test.go beyond "FAIL model endpoint" prefix to pin the exact detail strings through the delegation | ⚪ | S | Tests |
| 38 | Add lesson to crush-config `references/lessons.md` (by commit): parallel sessions + auto-commit on one tree = interleaved authorship; check git log first | 🟡 | S | Process |
| 39 | Refresh a2ui coverage/benchmark baselines (2026-08-18) post-restoration | ⚪ | M | Quality |
| 40 | Short fuzz campaign over the restored `RawMessage` decode paths (targets already exist) | ⚪ | S | Tests |
| 41 | Run full verification matrix steps 7–9 (`nix run .#test`, `nix build`, `nix flake check`) post-restoration | 🟠 | M | Quality |
| 42 | Rename `gatedRunner` → `healthGatedRunner` (grep-ability) or accept as-is | ⚪ | S | Cleanup |
| 43 | Refactor `openConfiguredPipeline`'s 5-value return into a small result struct | ⚪ | S | Refactor |
| 44 | Document why doctor (10s) and gate (5s) probe timeouts differ, in one place | ⚪ | S | Docs |
| 45 | CHANGELOG entry for: health gate, seventh restoration, ErrorKind completion (if repo keeps one) | ⚪ | S | Docs |
| 46 | docs-health VERIFY over the two 2026-10-31 reports next session (they contain cross-references to each other) | ⚪ | S | Docs |
| 47 | Consider documenting Q8_0 vs smaller-quant tradeoff for faster llama-server recovery (first-start is ~10 GB) | ⚪ | S | Docs |
| 48 | Add race test asserting passes never overlap (design guarantee, currently implicit) | ⚪ | S | Tests |
| 49 | Keep the samber-do skill's audit checklist result in the repo (link the report from the skill's case-study section, if the skill accepts consumer links) | ⚪ | S | Docs |
| 50 | Schedule the next status report after the R1 gate lands — measure whether recurrence class 8 exists (it should not) | 🟡 | S | Process |

## g) Three questions I cannot answer myself

**g1 — Fold semantics (blocks item 12, the only red test and the last lint finding):**
When the journal contains an event type this binary doesn't know (schema evolution, forward compatibility), should `ApplyViewState` **tolerate-and-ignore** (the test's expectation) or **fail the fold** (the current code)? I cannot decide this: it trades journal-safety (strict = corrupt/new-format data can't silently vanish reviews) against upgrade-robustness (tolerant = newer journals open on older binaries). What is the intended upgrade story for visionreviewd — must old binaries read new journals?

**g2 — Auto-commit daemon policy (blocks item 1, the recurrence class-killer):**
May I gate the auto-commit daemon itself (refuse to commit unless tree-wide v2-grep + pinned lint pass), or do you prefer a different design — e.g., converting it to open a branch/PR instead of pushing master directly? It is your infra tool; both options change your daily workflow, and I won't touch it without your call.

**g3 — Health-gate operator surface (blocks items 14/17):**
Should the gate be operator-visible configuration (`healthGate.enabled`, `probeTimeout` in config.json; `once --skip-unhealthy`), or do you want it always-on and invisible ("if the endpoint is down, skipping is always correct")? I can argue either side: configurability helps slow/odd endpoints and cron users; invisibility keeps the config honest and the behavior uniform.

---

**Per instruction: WAITING FOR FURTHER INSTRUCTIONS.** Section (f) is HARVEST-ready; say the word and it routes into TODO_LIST.md / ROADMAP.md.
