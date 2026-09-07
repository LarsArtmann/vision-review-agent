# Status Report: go-cqrs-lite Full Dependency Bump

**Date:** 2026-09-07 17:21 CEST
**Session scope:** Update ALL go-cqrs-lite sub-modules in vision-review-agent to LATEST published versions (user request: "Update to LATEST /home/lars/projects/go-cqrs-lite/**")
**Skill followed:** go-ecosystem-upgrade protocol (Phases 0–5)
**Branch:** master @ `91ca0d2` (clean; auto-commit daemon committed all session work)

---

## Executive Summary

All 14 consumed go-cqrs-lite sub-modules now ride their latest **published** tags
(decider v4.5.0, event v4.9.0, storage/bbolt v4.1.0, command v4.8.1, query v4.7.1,
metadata v4.6.0, record v4.4.0, snapshot v4.4.0, kv v4.2.1, dispatcher v4.3.1,
+ new indirect `storage/backuptest/v4 v4.1.0`). Upstream split `codec/v4` and
`flightrecorder/v4` into standalone `go-codec` / `go-flightrecorder` modules.

The daemon's `Store` was migrated off upstream-deprecated APIs
(`Repository.Load`/`Execute` → `LoadRef`/`ExecuteRef` with `id.NewStreamRef`;
pair form is removed in upstream v5). One pre-existing lint breakage (stale
`//nolint:recvcheck` from commit `2934585`) was fixed on sight. `vendorHash.nix`
updated per its documented procedure.

**Every gate is green:** build, vet, gofmt, golangci-lint (0 issues),
`go test -race ./...` (9 pkgs, run pre- AND post-migration), both json regimes
(jsonv2 full module; `GOEXPERIMENT=none` SDK subset), `go mod verify`,
`go mod tidy -diff`, `nix build .` + `nix build .#visionreviewd` + version smoke,
and all 73 internal/reviewd event-sourcing specs.

On-disk journal compatibility across the bump was **verified by inspection**:
bucket names and the `serializableEvent` wire struct/tags are byte-identical
between bbolt v4.0.0 and v4.1.0; record-layer changes are additive fields.
Existing journals load unchanged.

### Stat Cards

| Metric | Value |
|---|---|
| Sub-modules bumped | 11 upgraded + 1 added + 2 replaced upstream |
| Tests (full race suite) | 9 pkgs ok, 0 fail, 0 races (×2 runs) |
| reviewd event-sourcing specs | 73 PASS, 0 FAIL/SKIP |
| Lint issues after session | 0 (started post-bump at 3) |
| Canonical verification matrix | 7 of 8 steps run (`nix flake check` skipped) |
| Commits by auto-daemon | 3 (`2c196fa`, `166b5c8`, `91ca0d2`) |
| Journal format breaks | 0 (proven by wire-struct diff) |

---

## a) FULLY DONE

1. **Pre-flight classification** — enumerated all 14 sub-modules with current vs
   latest published versions (`go list -m -versions` per module); confirmed no
   `replace` directives, no `vendor/`, no `go.work`; confirmed local go-cqrs-lite
   master is ahead of its published tags and correctly decided to track
   **published proxy tags**, not local unreleased state.
2. **Baseline before migration** — `go build ./...` + `go test ./...` green
   (9 pkgs ok, 0 fail) under the jsonv2 default regime.
3. **The bump itself** — `go get @latest` for all 14 sub-modules, `go mod tidy`,
   `go mod tidy -diff` clean, `go mod verify` ("all modules verified").
4. **Breaking-API migration** — `internal/reviewd/store.go:180,226` migrated to
   `LoadRef`/`ExecuteRef` + `id.NewStreamRef`; verified upstream signatures in
   the module cache before editing.
5. **Lint debt fixed** — SA1019 deprecation warnings eliminated by the
   migration; stale `//nolint:recvcheck` directive removed
   (`internal/reviewd/config.go`); `golangci-lint fmt` fixed the golines
   line-length nit. Final: **0 issues**.
6. **Full verification** — build/vet/gofmt/golangci-lint clean; `go test -race
   ./...` green both before and after the store.go migration; jsonv2 regime full
   module; `GOEXPERIMENT=none` SDK subset build+vet+test; 73/73 reviewd specs.
7. **Nix leg** — vendorHash updated exactly per `vendorHash.nix`'s documented
   procedure (ran failed build, copied `got:` hash, re-ran); `nix build .` and
   `nix build .#visionreviewd` both succeed; daemon binary version smoke passes
   (defends against the silent-empty-build class).
8. **Journal wire-format compatibility proof** — diffed old vs new module cache
   sources: bucket names (`cqrs_events`, `cqrs_journal`, ...) unchanged;
   `serializableEvent` JSON tags byte-identical; `DecodePayloadAuto` error-wrap
   change confirmed invisible (consumer never matches error families).
9. **go.mod truth check** — every sub-module line re-verified against expected
   versions by grep (not trusted from `go get` output).
10. **Memory updated** — AGENTS.md daemon section gained a durable entry:
    bump facts, the Ref-form migration, and the journal-compatibility finding.
11. **Coherence check of auto-daemon commits** — verified `2c196fa`
    (go.mod/go.sum) and `166b5c8` (store.go/config.go) contents match the
    session timeline; final tree is exactly what was verified green.

## b) PARTIALLY DONE

1. **Canonical verification matrix — 7/8.** Skipped `nix flake check` (module
   eval checks for the NixOS module). Cheap to run; not run for time.
2. **Journal compatibility — inspection, not automation.** The "old journals
   load unchanged" claim rests on manual source diffs. No golden-fixture test
   pins the old wire format against future upstream changes.
3. **Lint baseline — inferred, not measured.** Baseline covered build+test only.
   Post-bump I found 3 lint issues and classified 2 as bump-caused (provable:
   they reference the new deprecations) and 1 as pre-existing (solid inference:
   I never touched config.go), but I never ran pre-bump lint to *prove* the
   split. Master had red lint between `2934585` and my fix and the baseline
   should have caught that.
4. **Report harvest — pending by design.** Section (f) below is the input for a
   `docs-health` HARVEST into TODO_LIST.md/ROADMAP.md; deliberately not run
   because the user said "then wait for instructions".
5. **Brutal self-review — folded into this file.** The brutal-self-review
   skill's canonical output is a styled HTML report at `docs/reviews/`; the
   user explicitly requested one Markdown status file instead, so the
   self-review lives in sections d/e/g here.

## c) NOT STARTED

1. **go-cqrs-lite release** — local master is 5–24 commits ahead of every
   published sub-module tag (real hardening fixes across 17–18 modules,
   e.g. `fix: wave of correctness/hardening fixes across 18 modules`). None of
   that is consumed by this repo. Releasing there is a separate go-release
   lifecycle task.
2. **Golden-journal fixture test** (see b2).
3. **CI lint-gate investigation** — why did `2934585` (lint config rework)
   land with `golangci-lint run` red? Root cause unknown; nothing was checked.
4. **v5 migration tracking** — upstream deprecations (pair-form Load/Execute,
   `CausationID`/`ActorID`/`ClientCreatedAt` fields) are slated for removal in
   go-cqrs-lite v5; no tracking item exists yet.
5. **Pre-existing tracked work** — a2ui builder art-dupl scan (already on
   TODO_LIST) was untouched; out of scope this session.

## d) TOTALLY FUCKED UP

Nothing in the shipped state is fucked up — the tree is fully green. Honest
candidates, ranked:

1. **My own verification one-liner lied mid-report.** The first go.mod version
   check used a regex (`v[0-9.]+$`) that can't match ` // indirect` lines and
   reported 9 false MISMATCHES. Caught and corrected immediately, and the
   skill explicitly warns to validate extraction against a known case first —
   I validated only the direct-dep pattern. A false alarm in a dependency
   migration report is exactly the trust-erosion the protocol exists to avoid.
2. **A mid-task state was committed unverified.** The auto-commit daemon landed
   `2c196fa` (go.mod/go.sum) between my tidy and my migration edits. By
   timeline reconstruction that commit builds (I had run build+vet after tidy,
   before editing store.go), but I never *proved* that exact commit green.
   Risk realized: none. Process hole: real.
3. **The baseline gap (b3)** — the closest thing to F11 in this run. The
   repo's master was carrying red lint from the previous commit and my
   baseline didn't expose it; I found it only after the bump and had to
   reason backwards about ownership.

## e) WHAT WE SHOULD IMPROVE

1. **Baseline = build + test + lint.** Always. Lint is in the canonical matrix;
   running it pre-bump would have surfaced the stale-nolint issue as
   pre-existing with evidence instead of inference.
2. **Data-compat claims need fixtures, not eyeballs.** "I diffed the struct"
   becomes false the next time someone bumps without diffing. A
   golden-bytes journal test converts this session's inspection into a
   permanent guarantee.
3. **Validate verification scripts before trusting their output.** Run the
   check against one known-good case (e.g. a direct dep with `// indirect`
   present) before reporting results.
4. **Label anomalies at the moment they appear.** The MISMATCH block went out
   raw; it should have been prefixed "my check regex is suspect" in the same
   breath.
5. **Restart the LSP after multi-file migrations.** Stale gopls/golangci_ls
   warnings polluted every subsequent tool output this session; CLI-over-LSP
   discipline saved correctness but the noise cost remained.
6. **Consider daemon interleaving.** The auto-commit daemon committing between
   bump and migration is expected behavior here, but mid-verification commits
   mean master can hold states no one verified. At minimum: after any daemon
   commit lands mid-task, re-run the fast gate (build+lint) on the new HEAD.
7. **Persist run artifacts durably.** Baseline/test outputs went to
   `/tmp/*.txt` — the skill says runner results belong in the repo. Ephemeral
   by default; a `docs/status/artifacts/` or flake app habit would fix it.

## f) TOP 50 THINGS TO GET DONE NEXT

*Near-term = do before the next release; ROADMAP fuel = schedule, don't commit.*

| # | Item | Priority |
|---|---|---|
| 1 | Run `nix flake check` to close the 8/8 matrix for this bump | Near-term |
| 2 | Golden-journal fixture test: old wire-format bytes load + fold correctly under v4.9 stack | Near-term |
| 3 | Investigate why `2934585` landed with red lint; make lint a required CI check | Near-term |
| 4 | Release go-cqrs-lite (tag the 5–24 advanced commits per module), then re-bump consumer to the hardening fixes | Near-term (sibling repo) |
| 5 | HARVEST this report's list into TODO_LIST.md / ROADMAP.md (docs-health) | Near-term |
| 6 | Cut vision-review-agent v0.7.0 (version var is `0.7.0-dev`; this bump should ride a release) | Near-term |
| 7 | Add `govulncheck` to the verification matrix (not currently there; new dep set unscanned) | Near-term |
| 8 | Full version-surface audit per go-ecosystem-upgrade `version-surface.md` (this session grepped docs/CI/nix lightly) | Near-term |
| 9 | Wire a `visionreviewd backup` subcommand using upstream bbolt backup lifecycle (v4.1.0 added backup tests upstream) | Near-term |
| 10 | `visionreviewd doctor` extension: probe journal readability with current lib version at startup | Near-term |
| 11 | Add "baseline must include lint" to AGENTS.md bump procedure notes | Near-term |
| 12 | Persist bump-verification loop as `scripts/` or flake app instead of ad-hoc shell one-liners | Near-term |
| 13 | Make vendorHash update a flake app (`nix run .#update-vendor-hash`) — kill the manual grep-got procedure | Near-term |
| 14 | Compile-time assertion/test pinning `serializableEvent` tags; fail loudly when upstream changes the wire | Near-term |
| 15 | Evaluate upstream `WithBatchCommit` for the daemon write path (new in bbolt v4.1.0; group commit = fewer fsyncs) | Near-term |
| 16 | Verify CI's no-jsonv2 job still excludes exactly the daemon dirs (go-cqrs-lite must remain daemon-only import) | Near-term |
| 17 | `go test -count=1 ./...` full sweep (cache-free) once, to back the green claim without cache benefit | Near-term |
| 18 | Confirm CHANGELOG entry exists for the bump if CHANGELOG.md exists | Near-term |
| 19 | Fix gopls `writestring` finding at `internal/reviewd/prompts.go:88` (LSP-only warning; decide fix vs ignore) | Near-term |
| 20 | Track section (g) questions to explicit resolution | Near-term |
| 21 | Pre-plan go-cqrs-lite v5 migration (upstream documents removals: pair-form APIs, CausationID/ActorID/ClientCreatedAt) | ROADMAP |
| 22 | Ask upstream for per-submodule CHANGELOGs (F6: incomplete changelogs force source-grepping every bump) | ROADMAP |
| 23 | Upstream contribution: wire-format contract test so bbolt minor bumps can't silently change tags | ROADMAP |
| 24 | Renovate/dependabot config for grouped go-cqrs-lite submodule bumps | ROADMAP |
| 25 | Dependency-freshness cron: `go list -m -versions` diff vs go.mod, report drift | ROADMAP |
| 26 | a2ui builder art-dupl scan (pre-existing TODO_LIST item) | ROADMAP |
| 27 | Benchmark pass duration before/after bump (event v4.9 journal_middleware, bbolt writeTx) — measure, don't assume | ROADMAP |
| 28 | Load test: 10k-event journal through pass+replay for perf sanity after bbolt refactor | ROADMAP |
| 29 | Fuzz `DecodePayloadAuto` seeded with old-version-produced payloads | ROADMAP |
| 30 | Evaluate new capabilities in query/v4.7, snapshot/v4.4, metadata/v4.6 the daemon might adopt (bumped blind as indirects) | ROADMAP |
| 31 | Document the codec/v4 → go-codec upstream split in a DEPS doc for future bumpers | ROADMAP |
| 32 | Update docs/DOMAIN_LANGUAGE.md if upstream introduced vocabulary we adopt (journal middleware, batch commit) | ROADMAP |
| 33 | Move the `/mnt/buildcache` gotcha into a bootstrap script (env bit me this session via GOMODCACHE) | ROADMAP |
| 34 | docs-health VERIFY pass on the new AGENTS.md claims (second-reader the compat claim) | ROADMAP |
| 35 | CI: full-module jsonv2 `-race` job alignment check with the canonical matrix | ROADMAP |
| 36 | `go mod tidy -diff` + `go mod verify` presence check in CI (matrix says mirrored; prove it) | ROADMAP |
| 37 | Compat-tagged integration tests (`//go:build compat`) with per-minor upstream fixtures | ROADMAP |
| 38 | Auto-daemon policy idea: fast gate re-run when a daemon commit lands mid-session | ROADMAP |
| 39 | journal_middleware (new in event v4.9): audit whether daemon wants mid-journal hooks | ROADMAP |
| 40 | Store schema_version strategy doc for future wire evolution | ROADMAP |
| 41 | Grep-guard test: no deprecated pair-form `repo.Load(`/`repo.Execute(` calls return | ROADMAP |
| 42 | LSP hygiene habit: restart gopls after multi-file migrations | ROADMAP |
| 43 | Stop writing session artifacts to /tmp; durable location for run outputs | ROADMAP |
| 44 | Re-check examples/ build under no-jsonv2 after every future bump (keep in matrix) | ROADMAP |
| 45 | Consider `visionreviewd journal stats` (sizes, event counts per stream) leveraging ReadAll | ROADMAP |
| 46 | Dependabot-style PR body: link upstream diff range for every dep bump | ROADMAP |
| 47 | Add `go.sum` + `vendorHash.nix` consistency check to CI (nix build dry-run job) | ROADMAP |
| 48 | Schedule periodic full canonical matrix runs (not only pre-release) | ROADMAP |
| 49 | Error-message audit: `DecodePayloadAuto` failures in replay paths could name the offending journal path | ROADMAP |
| 50 | Consider consuming upstream's new test helpers (commandtest store_suite) for reviewd E2E specs | ROADMAP |

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Deployment reality of visionreviewd journals:** are there real
   installations whose on-disk journals must survive upgrades long-term
   (dictating golden fixtures, a backup subcommand, and a doctor journal probe
   — items #2/#9/#10), or is wipe-and-recapture acceptable, making the
   inspection-level proof sufficient?
2. **Why did commit `2934585` land with red `golangci-lint run`?** I can see
   the commit broke lint; I cannot see whether CI didn't run, wasn't
   required, or was ignored. The fix (#3) depends on which.
3. **Release cadence for go-cqrs-lite:** should its unreleased master
   (hardening fixes across 17–18 modules) be tagged and re-bumped here as an
   immediate follow-up (#4), or do we deliberately ride published tags until
   a scheduled release? That's the sibling repo's lifecycle call, not mine.

---

## Verification Evidence (this session)

| Check | Command | Result |
|---|---|---|
| Baseline build/test | `go build ./... && go test ./...` | PASS (pre-change) |
| Post-bump build/vet/fmt | `go build/vet ./...`, `gofmt -l .` | PASS / clean |
| Race suite (×2: post-get, post-migration) | `go test -race ./...` | 9 ok, 0 fail |
| Lint | `golangci-lint run ./...` | 0 issues (3 found, all fixed) |
| jsonv2 regime | default env (`GOEXPERIMENT=jsonv2`) full module | PASS |
| no-jsonv2 regime | `GOEXPERIMENT=none` over SDK subset | build+vet+test PASS |
| Module hygiene | `go mod verify`, `go mod tidy -diff` | verified / clean |
| go.mod truth | grep all 13 module lines vs expected | all OK |
| Nix | `nix build .` + `.#visionreviewd` + version smoke | PASS |
| Delivering layer | `go test -v -race ./internal/reviewd/` | 73 PASS |
| Wire compat | diff bbolt v4.0.0 vs v4.1.0 sources | buckets + tags identical |
