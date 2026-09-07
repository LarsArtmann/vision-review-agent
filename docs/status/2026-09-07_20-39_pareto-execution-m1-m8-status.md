# Pareto Execution Status — Plan Run Interrupted at M8

**Created:** 2026-09-07 20:39 CEST
**Branch:** `master` @ `a1c95dc` (auto-commit) — **HEAD DOES NOT BUILD** (see d1)
**Base:** Plan `docs/planning/2026-09-07_17-28_pareto-post-bump-execution-plan.md`
(M1–M24, full execution authorized). Session start state: `8590dda`, plan just
delivered, nothing executed.

Format note: the status-report skill's canonical output is a styled HTML
dashboard; the instruction for this run explicitly names `.md`, so the user's
format wins and this file is Markdown. One-off override, not a new default.

---

## a) FULLY DONE

1. **M1 — Verification matrix closed 8/8 with evidence** (`556065b`).
   `nix flake check` all checks passed (incl. NixOS module evals + version
   smoke), `-count=1` and `-race` sweeps green (9 pkgs), both json regimes
   green, `go mod verify`/`tidy -diff` clean, lint 0 issues, nix builds +
   smoke ok — all at `8590dda`. Status report + AGENTS.md annotated.
2. **M2 — Golden-journal fixture regression suite** (`df4e7a2` + daemon
   commits for fixture/go.mod). Frozen 64 KiB journal
   (`internal/reviewd/testdata/golden-journal.bbolt`, 2 streams, all 3 event
   types, provenance + regen recipe). Five tests pin: decider fold, payload
   decode via stamped codec, raw CBOR envelope key set
   (`schema_version=1`, `encoding=cbor` — a detail source inspection had
   missed), payload JSON tags via reflection, plus `FuzzGoldenJournalDecode`
   (~1M execs clean, 967,548 execs in 32 s at baseline). The journal-compat
   claim is now machine-pinned, not eyeballed.
3. **M3 — v0.7.0 released end-to-end.** CHANGELOG `0.7.0` section; version
   vars flipped in both binaries (`5d12801`); annotated tag `v0.7.0` pushed;
   proxy propagated; **fresh-module `go get …@v0.7.0` validated**; GitHub
   Release created with curated notes (`--latest`); `0.8.0-dev` cycle opened
   (`f0ff43c`). Gate discipline held: CI green on the exact tagged commit.
4. **M4 — Red-lint root cause + CI gate.** Root cause: CI ran golangci-lint
   **v2.12.2** (action pin) while local ran **v2.13.2**; the older linter
   flags the standard `UnmarshalJSON`-pointer/`MarshalJSON`-value receiver
   pattern on the daemon config, so removing the stale `//nolint` (based on
   the local linter) turned CI red. Fixes: CI pin → `v2.13.2` in BOTH the
   action input and the config-verify `go install` (`c8e5e4b`); branch
   protection enabled with required status checks — first 7, then **9**
   contexts (`govulncheck`, `dep-drift` added) — verified working ("Bypassed
   rule violations" on admin pushes, required for PRs); AGENTS.md section
   records the rule "bump local linter ⇒ bump CI pin in same commit"
   (`5182cba`).
5. **M6 — Vulnerability scanning + dependency-drift visibility** (`17d805f`,
   `ddcf1ba`). govulncheck baseline: **0 reachable vulnerabilities**; module
   level: 2 ssh-DoS advisories in `x/crypto` cleared by bump to **v0.56.0**;
   1 unfixed advisory remains (GO-2026-5932, unmaintained openpgp — not
   called by our code). `scripts/check-deps.sh` + flake app `.#dep-drift`
   (validated: surfaced drift in fantasy v0.43.1, catwalk v0.52.28, testify,
   gomega); govulncheck CI job (needs `GOEXPERIMENT=jsonv2` for package
   loading — second commit fixed that) + weekly `Scheduled Security`
   workflow; vendorHash updated (`h0/BhZs…`); AGENTS.md matrix extended to
   9 steps + hygiene paragraph.
6. **M7 — `visionreviewd backup` subcommand** (`d98d091` + `a1c95dc` tail).
   Consistent journal snapshot via single bbolt read transaction
   (`tx.WriteTo`); `Store.Backup` + `BackupJournalFile` (raw read-only bbolt
   with bounded lock timeout `DefaultBackupLockTimeout=5s`); `JournalPath`
   helper dedupes 3 path constructions; CLI `backup [-config] [-wait] OUT`
   wired with usage + doc comment; tests: restore-into-working-store,
   held-journal fast-fail (100 ms), event-order parity, CLI E2E
   seed→backup→restore→**replay byte-identical**; README + AGENTS updated.
   Lint 0 issues, race green.
7. **F5.4 — a2ui art-dupl split brain closed by execution, not bookkeeping**
   (`c137687`). Ran the scan instead of adding a TODO: a2ui has 41 clone
   groups, all non-actionable/suppressed; the only actionable pair repo-wide
   (repeated analysis error-exit blocks in `cmd/vision`) extracted into
   `failAnalysis`. Repo is at **0 actionable clone groups**; AGENTS.md claim
   updated with fresh evidence and no dangling TODO_LIST reference.
8. **ROADMAP freshness** (part of M5): near-term paragraph no longer claims
   "CI green since 2026-08-17" (false since today's findings); records the
   v0.7.0 outcome and 9-check CI gate; open question #5 (release cadence)
   annotated with what still gates the sibling-repo decision.
9. **PR #1 (grpc 1.83.1 dependabot) unblocked**: stale checks refreshed,
   lint now passes on PR branch, vendorHash fixed on the branch (`1b1bf46`),
   11/11 checks passing as of 20:39 (merge state still recomputing).

---

## b) PARTIALLY DONE

1. **M8 — doctor journal-readability probe: ~70% done, AND IT IS WHAT LEAVES
   HEAD RED.** `VerifyJournalEvents` (pure fold-verify with
   stream/event-named errors) and `JournalLockHeld` (bounded-lock probe)
   exist in `internal/reviewd/store.go` and build+test green. The doctor
   wiring `checkJournal` in `cmd/visionreviewd/commands.go` is written but
   **references `errors.Is` without the `errors` import** —
   `cmd/visionreviewd` does not compile at HEAD (`a1c95dc`). Not tested, not
   linted, not documented.
2. **M5 — HARVEST: stable parts done, final sweep pending.** Split brain
   (F5.4) closed, ROADMAP refreshed, Q3 routed. NOT done: routing the
   surviving post-execution items into `TODO_LIST.md` (user-action items
   M19/M21/M22 enrichments, M15 outcomes) and annotating the plan document
   with per-task `done at <hash>` markers.
3. **M15 — Upstream capability evaluation: discoveries made, no decision
   note.** Found and verified: upstream `OpenWith` **cannot open read-only**
   (unconditional bucket creation collides with bbolt RO mode → always
   errors; and without a Timeout it hangs instead of erroring when locked).
   That is why backup uses raw bbolt. WithBatchCommit / journal_middleware /
   query-snapshot-metadata API review: not started; no ADR-style note in
   DEPS.md.
4. **M16 — CI alignment: 2 of 4 remaining checks done.** Already present and
   verified: no-jsonv2 job excludes exactly the daemon dirs; jsonv2 job runs
   `-race`. New this session: govulncheck + dep-drift jobs. MISSING:
   `go mod tidy -diff` + `go mod verify` CI job; go.sum↔vendorHash
   consistency job. (My session todo list marked M16 "completed" — that was
   wrong; this report corrects it.)
5. **M11 — Docs: AGENTS rules done (lint pin, 9-step matrix, hygiene,
   baseline-lint rationale), but `docs/DEPS.md` (codec split, wire contract,
   schema_version note, bump procedure) not written; DOMAIN_LANGUAGE check
   not done.**
6. **M22 — json/v2 external fix: friction is now documented twice (AGENTS
   govulncheck env note) but the go-auto-upgrade exclusion draft and
   proposal do not exist.**
7. **M23 — Replay error context: the _spirit_ landed in the doctor probe
   (`VerifyJournalEvents` names event index, type, and stream), but the
   Replay paths themselves still wrap `DecodePayloadAuto` errors without
   stream/journal context.**
8. **CHANGELOG discipline for the new cycle: `0.8.0-dev` has no
   `[Unreleased]` entries yet for backup (M7) or doctor probe (M8)** — the
   repo rule says completed TODOs/plan items get recorded there; neither
   landed.

---

## c) NOT STARTED

1. **M9** — Release go-cqrs-lite master (5–24 unreleased commits per module)
   and re-bump consumer. Gated on the sibling repo (user-owned) and on the
   cadence answer; everything needed to execute it is now proven (this
   session's bump method + golden fixture would catch any drift).
2. **M10** — `update-vendor-hash` flake app + persisted `verify-bump.sh`
   (validated against a known case). Today's manual vendorHash dance happened
   twice — the tooling would have saved two round trips.
3. **M12** — v5 migration tracking (ROADMAP entry) + grep-guard test against
   pair-form `repo.Load(`/`repo.Execute(` calls.
4. **M13** — Upstream asks (serializableEvent contract-test issue,
   per-module CHANGELOG request). Requires `verify-before-filing` first.
   Today found a THIRD candidate: `OpenWith` ReadOnly is unusable.
5. **M14** — Perf evidence: Pass/Replay benchmarks + 10k-event journal load
   test + baseline doc.
6. **M17** — Go 1.26.6 toolchain re-probe (5 stdlib CVEs; blocked on
   nixpkgs; last probed 2026-08-18).
7. **M18** — `internal/reviewd/prompts.go:88` WriteString concat (gopls
   finding, still live).
8. **M19** — DiscordSync 216-view watch: durable config + interval proposal
   (user cadence call).
9. **M20** — llama `--image-min-tokens 1024` evaluation (needs dev
   llama-server restart — user environment).
10. **M21** — SystemNix activation dry-run prep (docs currency check,
    doctor against template config, handoff checklist).
11. **M24** — Session hygiene bundle (durable artifact location,
    version-surface audit, commandtest eval, examples re-check note,
    bump-PR body template, LSP-restart habit).

---

## d) TOTALLY FUCKED UP

1. **Master HEAD (`a1c95dc`) does not compile.** The auto-commit daemon
   committed my mid-implementation `checkJournal` (`errors` import missing).
   Direct pushes bypass required checks (by design, admin), so nothing
   stopped it. One-line fix (`"errors"` import) + test + lint is THE first
   next action. Root lesson: auto-commit daemon + half-finished edits = red
   master; either finish-then-report or stash WIP before reporting.
2. **Master CI was red for a week unnoticed** (found today, root-caused,
   fixed): the lint-version drift (v2.12.2 vs v2.13.2) plus no required
   checks plus the auto-daemon removing a nolint directive meant `df4e7a2`,
   `556065b`, `8590dda`, `d5c5c74` all show `failure` in CI history. The
   fix is systemic (pin + required checks), but the fact stands: a week of
   red landed on the release branch and the release was almost cut on top.
3. **My first backup design was wrong** — it claimed "safe against a live
   writer", which bbolt cannot do (single read-write handle). The design
   error was caught by my own test hanging (600 s timeout), then a probe
   proved upstream `OpenWith` RO is broken entirely. Rewrote to the honest
   semantics (daemon stopped, bounded `-wait`). No wrong code shipped, but
   the plan's F7.1 assumption ("using bbolt Backup API … while running")
   was never validated before implementation started.
4. **v0.7.0 shipped before M7/M8 exist** — per plan order that is correct
   (gates were M1+M2), but it means the GitHub Release notes advertise
   journal-format pinning while the _operational_ journal tools (backup,
   doctor probe) land only in 0.8.0. Acceptable tradeoff, worth knowing.
5. **Session bookkeeping error:** I flipped M16 to "completed" in the todo
   list while 2 of its 4 subtasks were still open. Self-caught during this
   report; corrected above.
6. **Pre-existing ghost (not mine, still unfixed):** v0.6.1 tag reports
   internal version "0.6.0"; v0.2.1/v0.3.0 tag anomaly — both documented in
   ROADMAP open questions, both still open.

---

## e) WHAT WE SHOULD IMPROVE

1. **Finish-before-commit discipline for WIP under the auto-commit daemon:**
   the daemon commits within minutes; a half-edited file lands on master
   red. Options: (a) build+vet loop after every file before moving on,
   (b) a pre-push local gate, or (c) accept it and always end sessions with
   a build fix commit. At minimum: never walk away from a multi-file change
   mid-signature.
2. **CHANGELOG as part of "done":** every M-task that ships behavior should
   append its `[Unreleased]` entry in the same commit; today only v0.7.0's
   section was curated.
3. **Validate environment assumptions with a 30-second probe before
   designing against them** (the bbolt lock/RO behavior would have cost 5
   minutes to discover and saved a redesign).
4. **Todo hygiene:** subtask-level todos for the fine-grained plan (116
   subtasks) is too coarse at M-level for sessions this long; the two
   mis-tracked items (M16 status, M8 "wired" claim) came from updating
   state from memory instead of from the tree.
5. **The lint-pin bump rule needs a companion:** when bumping the local
   golangci-lint, ALSO re-run the LSP so its stale diagnostics stop
   confusing the next session (LSP panel still shows pre-fix warnings from
   hours ago).
6. **dependabot PR handling:** pushing non-dependabot commits to its branch
   works but can desync its automation; prefer `@dependabot commit`-style
   rebases or accept-and-merge quickly once green.
7. **The 9 required checks give PRs a real gate now — consider a CI job
   that fails when `go build ./...` fails on ANY GOEXPERIMENT setting**
   (the jsonv2/no-jsonv2 split is the one dimension where local green is
   easiest to get wrong).

---

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

Priority-ordered; the top of the list is urgent, the tail is ROADMAP fuel.

1. Fix the missing `"errors"` import in `cmd/visionreviewd/commands.go` —
   master must build again (5-minute fix).
2. Finish M8: test `checkJournal` (healthy journal passes, truncated/
   corrupted journal fails with clear message, locked-journal skip path),
   lint, docs (README doctor example), commit with full message.
3. Add `[Unreleased]` CHANGELOG entries for backup (M7) and doctor probe
   (M8).
4. Merge PR #1 (grpc 1.83.1) once checks are green — the vendorHash fix is
   already on the branch.
5. Update vendorHash.nix if PR #1's merge shifts go.sum (squash merge will).
6. Run the full 9-step matrix after the M8 fix lands; record evidence.
7. HARVEST sweep (finish M5): annotate the plan document M1–M8 with
   `done at <hash>` markers inline.
8. HARVEST sweep: route surviving items into TODO_LIST (M19/M21/M22 user
   actions, a2ui/llama follow-ups) and ROADMAP (long-lived M12/M14/M15
   items).
9. M10a: `update-vendor-hash` flake app (automates the got:/specified:
   dance; used twice today manually).
10. M10b: `scripts/verify-bump.sh` (build+vet+lint+test+tidy-diff+verify)
    validated against current known-good state.
11. M12a: ROADMAP entry for go-cqrs-lite v5 migration trigger.
12. M12b: grep-guard test that no pair-form `repo.Load(`/`repo.Execute(`
    calls exist (protects the v5 migration).
13. M11a: write `docs/DEPS.md` — codec/v4→go-codec split, serializableEvent
    wire contract (CBOR envelope, schema_version=1), bump procedure.
14. M11b: schema_version/evolution note in DEPS.md (fixture proved rows
    stamp schema_version=1; document re-blessing procedure).
15. M15a: audit event/v4.9 `journal_middleware` — what hooks exist, adopt
    or decline.
16. M15b: spike `WithBatchCommit` write-throughput measurement on a test
    backend (read-only experiment, no write-path change).
17. M15c: write the adopt-or-decline ADR note (DEPS.md) covering M15a/b +
    query/snapshot/metadata API fit.
18. M16a: add `go mod tidy -diff` + `go mod verify` CI job.
19. M16b: add go.sum↔vendorHash consistency job (hash-math or nix dry-run).
20. M23: wrap Replay-path `DecodePayloadAuto` errors with stream + journal
    path context; update affected tests.
21. M13a: load `verify-before-filing`, then file the serializableEvent
    contract-test ask upstream (repro = our golden fixture method).
22. M13b: file/discuss per-module CHANGELOG request upstream.
23. M13c: report upstream `OpenWith` ReadOnly defect (hangs on held lock
    without Timeout; errors unconditionally even when unlocked — bucket
    creation ignores RO). Three concrete repros from today's session.
24. M18: fix `prompts.go:88` WriteString concat per strings.Builder
    pattern; verify lint/bench clean.
25. M17: re-probe locked + unstable nixpkgs for go 1.26.6 (5 stdlib CVEs);
    if shipped: go.mod + flake bump + full matrix.
26. M24a: decide durable location for run artifacts (or drop the habit).
27. M24b: full version-surface audit per go-ecosystem-upgrade reference.
28. M24c: evaluate upstream `commandtest` store_suite for reviewd E2E.
29. M24d: add note — re-check examples/ under no-jsonv2 after every bump.
30. M24e: bump-PR body template with upstream diff-range link.
31. M24f: session hygiene note — LSP restart after multi-file migrations.
32. M19a: fold the 216-view DiscordSync glob into a durable config
    (discover extension).
33. M19b: estimate full-pass duration; present cadence options (user call).
34. M21a: verify `docs/visionreviewd-systemnix.md` currency post-bump.
35. M21b: run `visionreviewd doctor` against the template config.
36. M21c: produce the SystemNix activation handoff checklist.
37. M22a: inspect go-auto-upgrade rule format for the json exclusion.
38. M22b: draft the `encoding/json` exclusion change + propose with the 4
    breakage dates as evidence.
39. M20a: restart dev llama-server with `--image-min-tokens 1024` (user
    environment) and rerun a2ui Generate on the dense-screenshot corpus.
40. M20b: compare fidelity both settings; write recommend/decline note.
41. M14a: benchmark Pass + Replay on a seeded store; capture baseline.
42. M14b: 10k-event journal generator + load test; record timings.
43. Bump fantasy v0.41.1→v0.43.1 (dep-drift surfaced it; needs its own
    verified change, likely API-affecting).
44. Bump catwalk v0.52.28 + testify/gomega/ginkgo minors (dep-drift list).
45. Watch GO-2026-5932 (openpgp advisory) for an upstream fix; document
    any new exposure if catwalk/fantasy pull it into reach.
46. Update FEATURES.md for backup subcommand + doctor journal probe
    (feature inventory discipline).
47. Add `visionreviewd backup` to the SystemNix docs as the pre-upgrade
    ritual (ties M21 to M7).
48. Consider promoting journal lock-wait telemetry (daemon log line when a
    pass skips a locked journal) — small observability win.
49. ROADMAP fuel: `backup --verify` (open the snapshot after writing and
    fold-verify it) as a future flag.
50. ROADMAP fuel: retention/GC now has a natural shape — backup first, then
    `visionreviewd gc --before <ts>` (prune journal + orphaned blobs).

---

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **go-cqrs-lite release cadence (gates M9):** the sibling repo's 14
   sub-modules are 5–24 commits ahead of their published tags. Do you want
   me to cut fresh tags for all of them and re-bump this repo now
   (I have the whole method proven from today), or wait until a concrete
   consumer need forces it? If yes: same-version-bump (v4.x.y+1) per
   module, or version-reset to a coordinated date-based scheme?
2. **Journal deployment reality (gates backup/doctor priority):** do
   real visionreviewd journals exist outside this repo's test/dev runs
   (e.g. on a host where you plan SystemNix activation)? If yes, M8/M7
   become pre-activation must-haves and I would add a backup-verify flag +
   document the stop-daemon/backup/start ritual in the SystemNix docs
   before you activate; if no, they stay hardening and I de-prioritize.
3. **0.x release presentation policy:** v0.7.0 is the first release marked
   `--latest` on GitHub (all previous v0.x were prerelease-marked, which
   is why v0.2.0 held the badge). Keep `--latest` for every future 0.x
   release, or reserve "Latest" for 1.0+ and go back to prerelease
   marking?

---

_Point-in-time snapshot. Annotate, never rewrite (docs-health ANNOTATE)._
