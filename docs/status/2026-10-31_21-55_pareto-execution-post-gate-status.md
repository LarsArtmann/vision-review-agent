# Post-Gate Status Report — Pareto Plan Execution, Trust Track Closed

**Session:** 2026-10-31 ~20:30–22:00 CET · **HEAD at report:** `4bb5d1a`
(plan input: `a6b0196`) · **Executor:** Crush session 2 (parallel to session 1)
**Plan:** `docs/planning/2026-10-31_18-15_pareto-trust-resilience-fleet.md`

## Executive summary

The plan's TRUST track (the 1% that delivers 51%) is **closed and
machine-verified**: the jsonv2 recurrence class is root-killed, guarded at
three independent layers, and the 26-day-red fold test is green with the
repo's last pinned-lint finding gone. Beyond it, this session + the
parallel session executed the RESILIENCE and PROOF tracks almost fully and
root-caused a **bonus find the plan did not contain**: CI has been red on
three required jobs (nix-flake-check, govulncheck, build-and-test) — all
three now fixed at root cause. Two concurrent sessions executing one plan
on one tree worked: every overlap was caught by end-state verification, and
the genuinely-duplicated work was complementary (each session fixed the
other's unlinted auto-commits).

## a) Trust track (T1a, T2, T3) — DONE, artifact-backed

- **T1a root-kill:** `.go-auto-upgrade.json` committed. Schema verified
  against `cmd/go-auto-upgrade/config.go` (per-project exclude arrays);
  invocation contract discovered and recorded (the tool has NO
  auto-discovery and NO systemd unit runs it — sessions must pass
  `--config`); A/B-proven with the tool itself (`--check` mode): with the
  config the 7/7 migrator run excludes jsonv1tov2, without it the whole
  tree is targeted.
- **T2 guards:** parallel session shipped package guards
  (`internal/reviewd` + `pkg/vision` incl. the a2ui walk) with positive
  controls; this session added `TestNoJSONv2ImportsModuleWide` (walks the
  whole module from reviewd, so a recurrence in `pkg/errors`, `cmd/*`, or
  `examples` fails ANY reviewd test run) + module-root discovery controls,
  and proved non-vacuity by seeding a raw-string v2 probe in `pkg/errors`
  → guard FAILED naming the file → probe removed → green.
- **T3 fold:** D1 = tolerate-and-ignore implemented (`UnknownEvents`
  counter on `ViewState`, matching the doc comment that always promised
  it); `TestApplyViewStateIgnoresUnknownEventType` green after 26 days;
  err113 gone; `internal/reviewd` linted 0 issues for the first time since
  the finding existed. (The parallel session independently landed the same
  decision + a decision record at `docs/decisions/`.)

**Recurrence #8 watch (T27.1): ZERO.** No v2 import has appeared since the
exclude file landed; the tree-wide grep is clean at report time.

## b) Resilience + proof tracks — DONE (jointly with the parallel session)

| Task | State | Evidence |
|------|-------|----------|
| T5 retry wiring | done (parallel wired; this session added the non-retryable-400 single-call test + example-config stanza) | `TestReviewerRetriesTransientFailure`, `TestReviewerDoesNotRetryNonRetryableFailure`; example config decodes to the default policy exactly |
| T6 watchdog | done (this session) | `internal/reviewd/notify.go` hand-rolled sd_notify (protocol-tested against a real unixgram socket), `WithServiceNotifier` daemon option, NixOS `Type=notify` + `watchdogSec` (30m default, corpus guidance documented) |
| T7 gate maturation | done (parallel: E2E die/skip/recover + observability; this session: D3 config stanza `healthGate` + `once -skip-unhealthy` + tests) | `TestHealthGateEndToEndDieSkipRecover`, `TestLoadConfigHealthGateStanza`, `TestRunOnceSkipUnhealthyFlagParses` |
| T11 gate observability | done (parallel) | `consecutiveSkips`/`lastHealthy` + deferred-captures in skip warnings |
| T12 doctor | done (parallel: `--json` + report shape; this session: blobDir probe = 6 checks now) | live doctor run against the new DiscordSync config |
| T25 pass observability | done (this session) | `PassResult.Pending` + `duration` in the pass log + overran-interval warning; `TestPassPendingCountsFailedReviews`, `TestRunPassWarnsWhenOverrun` |
| T18 quality trio | done (1: `TestDaemonPassesNeverOverlap` race pin; 2: parallel's per-glob scan isolation; 3: shared-http.Client DECLINED with rationale in ROADMAP) | |
| T17 test refresh | done | a2ui coverage **93.0%** (was 89), benches at-or-better than the 2026-08-18 baseline (Marshal 127.9µs, Validate 3.27µs, Unmarshal 211µs, Decompile 2.07µs), 5 fuzz targets × 60s clean, `defaultIDs` table test added |
| T9 HARVEST | done | TODO_LIST surgery + ROADMAP routing (ops ideas section); done items removed |
| T4 proof docs | done | CHANGELOG `[Unreleased]` (gate, root-kill, fold, watchdog, retry, observability, CI fixes, 7th restore), FEATURES health-gate row, AGENTS artifact-bearing claims, the 2026-10-05 report's two false "grep clean" claims annotated inline |
| T16 fad59ce audit | done | go.sum −92 = ALL `/go.mod` graph hashes (tidy residue; nothing needed dropped — `go mod verify` + builds green); SECURITY.md +40 = accurate standard policy, accepted |
| T20 no-jsonv2 scope | done (parallel, better than planned) | fantasy-free set proven EMPTY (pkg/errors → fantasy → kaptinlin), job redefined as a watchdog that is GREEN while the blocker persists and RED when it clears |
| T10 lint lockstep | done | CI pins → 2.14.0 (both sites, same commit); local 2.14.0 = 0 issues repo-wide; the 2.14.0-new wsl grouped-const finding fixed at source |
| T8 matrix | done at `4bb5d1a` | build/vet/gofmt clean; full race 9/9; `GOEXPERIMENT=none` failure = exactly the documented kaptinlin chain (and now provably unreachable-set-empty); `go mod verify` ok; `nix run .#test` pass, `.#lint` 0 issues; `nix build .` + `.#visionreviewd` ok; `nix flake check` ALL CHECKS PASSED; govulncheck 0 code-affected at go 1.27.3 |
| T13a KNOWN escalation | done (parallel) | KNOWN_ESCALATE_DAYS=7 + first-seen state in site-monitor.sh |
| T13b off-machine vantage | mechanism done, URL owner-side | `SITE_MONITOR_PING_URL` dead-man's switch in site-monitor.sh (verified: exactly one ping per clean run against a loopback listener; fleet is 18-ok right now); the actual healthchecks.io URL is an owner account step |
| T13c canonical timers | done | SystemNix home.nix defines site-monitor + fleet-review user units (deploy-gated; legacy symlinks stay until the switch, removal steps documented) |
| T15 DiscordSync config | done (D6=1h) | `~/.config/visionreviewd/discordsync.json` + repo copy; doctor-validated (232 PNGs match, dirs writable, fresh journal; only the not-yet-deployed model endpoint down) |
| T14.1 checklist refresh | done | visionreviewd-systemnix.md: 6 checks, `-json`, Type=notify/watchdog guidance |
| T23 lessons | done | crush-config `ee5018c`: pin-then-verify + numbers-re-derived (parallel-session lesson + LSP-cache lesson were already there) |
| T24 LSP | done | restarted; CLI remains the contract (documented as a lesson by the parallel session) |
| T26 release mechanics | done (D7=wait) | bump-PR template committed to docs/templates; dep state: go 1.27.3 floor, CI pins aligned |

## c) Bonus find: CI was red on THREE required jobs — all root-caused

The plan's errata said "branch protection carries 10 required checks" and
assumed CI green. It was not (since ~2026-10: three jobs red on every
master push):

1. **nix-flake-check** — treefmt's `goimports` loads packages via the go
   toolchain; in the network-less nix sandbox it tried downloading
   go1.27.1 and died. The log's "formatted 3 files (0 changed)" read like a
   formatting miss and was a red herring; the real fix is dropping
   goimports from treefmt (import order remains enforced by the `gci`
   linter — a required check; gofumpt needs no package loading). Verified:
   `nix flake check` → **all checks passed**.
2. **govulncheck** — `go 1.27.1` in go.mod pinned setup-go to a toolchain
   with ten open stdlib CVEs (GO-2026-66xx/6809 family, fixed in
   1.27.2/.3). Floor bumped to `go 1.27.3` (the flake's nixpkgs `go_1_27`
   already shipped 1.27.3 — nix builds were never affected). Verified:
   govulncheck at 1.27.3 reports 0 code-affected vulnerabilities.
3. **build-and-test (race)** — the two sessions' interleaved auto-commits
   mid-flight; the full race suite is 9/9 green at the report HEAD.

## d) Not done / owner-gated (unchanged or newly explicit)

- **T14.2/14.3 SystemNix deploy** (owner: push, `--update-input`, deploy,
  first-pass watch) — the module now also brings Type=notify + watchdog.
- **T13b URL** — the healthchecks.io (or equivalent) account + URL is an
  owner step; the mechanism is live and tested.
- **T22a 9-site deploy** — BLOCKED in this session: no `firebase` CLI on
  PATH, and 9 × (install+build+deploy) needs its own quiet window. Stays
  in TODO_LIST.
- **T22b fleet polish, T21 Wave-4** — plan-consistent: each item its own
  session (ROADMAP/TODO_LIST as routed).
- **D2 daemon gating** — PMA is owner infra; the repo-side root-kill +
  guards + CI grep make the repo safe regardless; recommendation on
  record.
- **D7 release** — decision "wait" honored; nothing tagged.

## e) Honest caveats

- Two sessions raced several files mid-edit (one transient build break in
  `scan.go`, one AGENTS.md mtime race); all resolved by end-state
  verification, and every claim in this report was re-derived from the
  tree at `4bb5d1a`, not recalled.
- The module-wide jsonv2 guard depends on `internal/reviewd` tests being
  run — a recurrence would evade it only if NO test run happened anywhere;
  CI's grep + depguard + the package guards cover that path.
- `/tmp` tmpfs hit 100% mid-session (accumulated per-command Go caches);
  ~20 GB of stale cache-shaped mktemp dirs were removed (Go-cache layout
  verified, >30 min old, tmpfs-ephemeral class); session caches moved to
  persistent `~/.cache/vra-{go,lint}cache`.

## f) Trust-track score (T27.1)

The plan's 51% question — *can master silently rot again?* — is now **no,
at three layers**: (1) root cause excluded for this repo, (2) `go test`
fails anywhere a v2 import returns (CI-bypass-immune), (3) CI grep +
depguard on every push. Residual risk is daemon-gating (D2, owner infra).
Score: trust track **complete**; resilience track complete except owner
deploy; proof track complete with this report as the closing artifact.

*Point-in-time snapshot per docs policy. Route follow-ups through
TODO_LIST.md, not this file.*
