# Dedup Session Status — Round 3 (the `-t 2` review pass)

- **Written:** 2026-09-22 22:52 CEST
- **Scope of this round:** user-pasted `art-dupl --sort total-tokens -t 2 --type-aware` output (4 clone groups) + "review!" instruction.
- **Lineage:** supersedes the _current-state_ sections of
  [`2026-09-22_22-38_dedup-session-status-round2.md`](2026-09-22_22-38_dedup-session-status-round2.md);
  rounds 1–2 history is not repeated here. Self-contained for the dedup thread; the pre-existing a2ui/lint/toolchain findings carry over as tickets.
- **Open questions:** the 3 questions from round 2 remain unanswered; restated in §g.

---

## Round-3 verdict table (what the user asked for)

| `-t 2` group                                                                              | Verdict                                                                                                                                          | Action taken                                                                                                                                                                                                                                                                                                                                                |
| ----------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GenerateOptions.applyDefaults` (generate.go:114) + `Compile` (surface.go:69) zero-checks | **Harmful semantic clone** — the SAME domain rule ("empty surface/catalog ID → `defaultSurfaceID` / `DefaultCatalogID`") maintained in two types | **Extracted `defaultIDs(surfaceID, catalogID string) (string, string)`** into surface.go:64; both call sites are now one-liners. This REVERSES the round-2 accept verdict — one rule in two places is not "intentional similarity". The round-2 rationale ("values already single-sourced constants") was wrong: the _rule_, not the values, was duplicated |
| `Generate` spec-fallback (generate.go:78-84)                                              | Not flagged, left untouched                                                                                                                      | Deliberate: different rule (caller-override fallback to `opts.SurfaceID`/`opts.CatalogID`), not part of the clone                                                                                                                                                                                                                                           |
| component/image/preprocess 5-line `if err != nil` tails                                   | Accept (unmarked)                                                                                                                                | Universal Go error-wrap idiom; three unrelated operations (child-list JSON encode, base64 decode, image.Decode), three packages                                                                                                                                                                                                                             |
| config.go + store.go error-wrap tails                                                     | Accept (unmarked)                                                                                                                                | Same idiom; config JSON encode vs bbolt journal read                                                                                                                                                                                                                                                                                                        |
| commands.go `newConfigFlagSet("compare"/"events", stderr)` call lines                     | Accept (unmarked)                                                                                                                                | The clone _is_ the round-1 extracted helper's call site (same class as `NewAgentFromArgs`); per-command flag registrations stay local on purpose                                                                                                                                                                                                            |

**Post-pass scan ladder (empirical, this round):** `-t 3` → **0 groups** · `-t 2` → **3 groups** (all idiom) · `-t 1` → **4 groups** (the 3 + NArg-usage ×2).

---

## Brutal self-review (the user's first three questions, answered directly)

### What did I forget?

1. **I did not re-run the widest scan (`-t 1`) before writing doc counts.** I encoded "6 clone groups at `-t 1`" into DUPLICATION_POLICY.md from arithmetic (7 − 1 extraction) and forgot the 2 previously `// art-dupl:accept`-marked groups also drop out of `-t 1` output. The real number was 4. Caught it because I re-ran the ladder for verification, then corrected the docs — but the wrong number briefly existed in a file whose whole job is to be the source of truth.
2. **No direct unit test for `defaultIDs`.** Coverage is indirect (TestCompile asserts catalog defaulting, generate BDD specs assert the Generate path). A 10-line table test (both empty / one empty / neither) was in reach while I was in the file and I skipped it.
3. **No full-module `go build ./...` after the edit** — I verified the touched package (gofmt/vet/lint/test) and the daemon (build+vet), which covers the blast radius, but the cheap global build would have removed all doubt.
4. **HARVEST is still not done** — round-2's (f) list was never routed into TODO_LIST/ROADMAP, and this round adds another (f) list behind it. Two unharvested reports in one day is how items get entombed.
5. **CHANGELOG entry for the dedup refactor (rounds 1–5)** still does not exist.

### What is stupid that we do anyway?

1. **Scanner counts are transcribed into docs by hand, twice now, and drifted both times** (the round-2 table said NArg ×3; the scanner sees ×2 because `compare` uses named `compareArgCount`). The accepted-groups table has no machine check. Docs that require a human to re-verify numbers will rot.
2. **The toolchain mismatch is documented instead of fixed.** go.mod requires 1.27.1; the flake devShell ships 1.26.7. AGENTS.md now contains a beautiful paragraph on how to survive the gap (explicit `GO=` prefix, empty-`$GO`-runs-shell-`test` trap) — that paragraph is a monument to a root cause nobody has removed. Every session pays the tax again.
3. **Stale gopls errors are tolerated across sessions.** The 4 `newConfigFlagSet` undefined errors are CLI-disproven phantom errors from round 1 that still render in every file view today. Trust in diagnostics erodes a little each time.

### What could I have done better this round?

1. Run the full ladder (`-t 1`, `-t 2`, `-t 3`) FIRST, then judge, then edit docs once, from evidence.
2. Added the `defaultIDs` table test in the same edit as the extraction.
3. Challenged the round-2 accept verdict _before_ the user's prompt surfaced the group again — the "review!" was the third time this pair was looked at; the first two passes accepted it wrongly.

### What could I still improve (going forward)?

1. Machine-verify the policy doc: a tiny script (or test) that re-runs the ladder and diffs the counts + checks `// art-dupl:accept` markers still exist at claimed sites.
2. Close the two broken a2ui tests at the root (sorted key emission + `omitzero`), per the round-2 ticket — master being red is the single worst fact in this report.
3. Fix the flake devShell and add a flake-vs-go.mod version guard, deleting the workaround paragraph from AGENTS.md.
4. Kill the gopls phantoms (restart, then escalate to crush-config if they persist).

### Direct answers to the remaining self-review questions

- **Did I lie to you?** No. But I _repeated unverified numbers_ (round-2's "7 groups") into this round's first doc edit — that is lie-adjacent and was corrected to 4 within the round.
- **Ghost systems?** None found. `defaultIDs` is wired into both consumers; nothing was created without a caller.
- **Split brains?** Three, all small: (1) DUPLICATION_POLICY.md vs AGENTS.md duplicate the group counts by hand — synced now, will drift again; (2) a2ui has TWO adjacent ID-defaulting rules (constant-fallback in `defaultIDs` vs caller-override in `Generate`) — legitimately distinct, but the relationship is only documented in this report and the policy table, not at the `Generate` site; (3) `GenerateOptions.SurfaceID`'s doc comment hardcodes `defaults to "main"` instead of naming `defaultSurfaceID`.
- **Scope creep?** Avoided. The pre-existing a2ui failures, the uncommitted `.golangci.yaml`, and the flake gap were all ticketed, not touched.
- **Did we remove something useful?** No. `defaultIDs` replaced duplicated logic; nothing else was deleted this round.
- **Tests?** a2ui green except the 2 known master breaks; gap = no direct `defaultIDs` test; gap = no automated check that the policy doc's scan claims are true.

---

## a) FULLY DONE

1. **All 4 `-t 2` clone groups judged** (read all 8 sites + researched the a2ui ID-defaulting flow across 37 constant references).
2. **`defaultIDs` extracted** — pkg/vision/a2ui/surface.go:64; `Compile` (surface.go:82) and `GenerateOptions.applyDefaults` (generate.go:114) converted; the fallback rule now has exactly one home.
3. **Round-2 accept verdict for the ID-fallback pair formally reversed** in docs, with the reasoning recorded (second pass on the same pair; first verdict was wrong).
4. **Scan ladder re-verified empirically:** `-t 3` = 0, `-t 2` = 3 (idiom-only), `-t 1` = 4 (idiom-only). Zero harmful clones remain.
5. **Verification suite on touched code:** gofmt clean · `go vet` clean · `golangci-lint run ./pkg/vision/a2ui/` → **0 issues** · a2ui test failures unchanged (only the 2 known pre-existing breaks) · daemon `go build` + `go vet` clean (CLI re-disproves the stale gopls errors).
6. **Docs corrected and synced:** DUPLICATION_POLICY.md (current state, helper-table row for `defaultIDs`, accepted-groups intro, NArg row fixed ×3 → ×2 with the `compareArgCount` explanation, elimination list extended to 4 groups) and AGENTS.md Code Duplication section (4 / `-t 2`: 3 / `-t 3`: 0).
7. **Root-cause note recorded** for why the scanner sees 2 of 3 NArg sites (named constant breaks token equality) — previously undocumented scanner behavior.

## b) PARTIALLY DONE

1. **a2ui Go 1.27 json/v2 wire regressions** — diagnosed (map-key order + `omitempty` ignored on `SendDataModel`), proven pre-existing (clean-worktree check, round 2), ticketed in TODO_LIST.md; the fix itself is not started. Master is red.
2. **Lint fallout of the uncommitted `.golangci.yaml` rule change** — all 11 findings enumerated and ticketed; none fixed; ownership of the config change still unknown.
3. **Toolchain alignment** — the mismatch (go.mod 1.27.1 vs flake devShell 1.26.7) is documented in AGENTS.md with a working command recipe, but the flake itself is unchanged.
4. **Docs health** — round-2 + round-3 (f) lists written but not harvested into TODO_LIST/ROADMAP; CHANGELOG entry for the dedup thread missing.
5. **Accepted-groups table trustworthiness** — corrected this round, but still hand-maintained and unverified by any machine check.

## c) NOT STARTED (known from this session's context; nothing new researched)

1. The a2ui determinism fix (sorted key emission in `Component.MarshalJSON` / `UpdateDataModel.MarshalJSON`).
2. `omitempty` → `omitzero` migration (or code-gating `SendDataModel`) + re-pinning `TestMessageWireShape`.
3. Flake devShell bump to 1.27.1 + flake-vs-go.mod version guard.
4. Answers to the 3 open questions (§g) — gating most of the above.
5. go-cqrs-lite v5 migration (ROADMAP item, untouched, out of this thread's scope).

## d) TOTALLY FUCKED UP

1. **Master is red and has been for days.** `pkg/vision/a2ui`: `ExampleCompile` (nondeterministic map-key order under native `encoding/json/v2` — 5 runs → 4 orderings) and `TestMessageWireShape/createSurface{,_with_theme}` (`omitempty` ignored on `SendDataModel bool`). Proven not caused by this session; still unfixed; every fresh clone of master fails `go test ./pkg/vision/a2ui/`.
2. **An uncommitted `.golangci.yaml` change with unknown ownership** (adds wsl_v5 / exhaustruct_v5 / gocognit / tagliatelle rules) is sitting in the tree, producing 11 findings. Nobody has claimed it, and CI's pinned v2.13.2 has not been checked against it. An anonymous config driving required CI gates is a process hole.
3. **The documented dev environment contradicts the build.** Every command this session needed the explicit 1.27.1-store-path prefix; the flake — the project's declared source of reproducibility — ships a toolchain that cannot build the module it defines.
4. **Docs encoded scanner output that was never re-verified** (group counts, NArg ×3). The source-of-truth doc lied by staleness twice in one day. Both fixed this round; the _mechanism_ (hand transcription) is still in place.
5. **This round's own blunder:** a wrong count ("6 clone groups") written into the policy doc before empirical verification — caught and corrected within the round, but it happened exactly the way the last round's lesson said not to.

## e) WHAT WE SHOULD IMPROVE

1. **Verify, then write.** Doc edits that state scanner/tool/test facts must be preceded by the command that produces those facts in the same round — counts from a previous round are not evidence.
2. **Machine-check hand-maintained truth.** The accepted-groups table (counts + marker presence) should be validated by a script or test, not memory.
3. **Fix root causes the same day they get a workaround paragraph.** The AGENTS.md toolchain-recipe paragraph should be deleted by _fixing the flake_, not maintained forever.
4. **Second-pass verdicts on accepted clones.** Round 2 accepted the ID-fallback pair; round 3's extraction proved acceptance wrong. Any group accepted with a "values are shared" rationale deserves one re-read before being encoded as permanent.
5. **Close red master before refactor rounds.** Deduping while the package under edit has known-broken tests means every verification run ends in a known-failure diff — workable, but it blurs the signal for exactly the packages most worth protecting.
6. **Harvest within the same session.** Two (f) lists now exist outside TODO_LIST; each unharvested report multiplies the next session's triage cost.

## f) Up to 50 things we should get done next

_Brainstorm, not commitment list (per status-report guidance). Ordered roughly by impact; first block is the highest-impact core._

**Wire-regression cluster (unblocks green master):**

1. Answer §g question (c) — byte-stable vs semantically-equal wire output; everything below keys off it.
2. Sorted/deterministic key emission in `Component.MarshalJSON` (the `map[string]any` props site).
3. Same for `UpdateDataModel.MarshalJSON`'s data-model map.
4. Migrate a2ui marshal tags `omitempty` → `omitzero` (Go 1.24+ native, json/v2-aware).
5. Alternative if tags stay: code-gate `SendDataModel` in `MarshalJSON`.
6. Re-pin `TestMessageWireShape` wants after the fix.
7. Re-pin `ExampleCompile` (or restructure it to be order-proof).
8. Add a marshal-twice byte-equality golden test (regression pin for determinism).
9. Run a2ui suite under `GOEXPERIMENT=none` and `GOEXPERIMENT=jsonv2` — both green.
10. Check `Decompile` and `Compile` round-trip paths for other map-order assumptions.
11. Add explicit null-vs-absent `SendDataModel` wire cases under both JSON regimes.
12. Re-record a2ui baselines (coverage %, benchmarks) — AGENTS.md numbers predate the 1.27 native json/v2 switch.
13. Consider a fuzz target for the marshal path under json/v2.

**Lint-fallout cluster:**
14. Answer §g question (b) — establish ownership/intent of the uncommitted `.golangci.yaml`.
15. 3× tagliatelle nolints for `sourceURLs`/`sourceURL` (openaicompat wire-format precedent exists).
16. 3× wsl_v5 findings — fix or justified nolint.
17. gocognit refactor of `TestGoldenJournalPayloadsDecode` (extract helpers, keep golden assertions).
18. 4× exhaustruct decisions: populate `Captured.SourceURL`/`Pipeline.sourceURLs` or nolint with reason; same for the 2 `bbolt.Options` sites.
19. Verify CI lint (pinned v2.13.2) agrees with local on the new rules before committing the config.
20. Commit or deliberately revert the `.golangci.yaml` change — end the anonymous-config state.

**Toolchain cluster:**
21. Answer §g question (a) — is the 1.27.1 bump intentional/finished?
22. Bump flake devShell to go 1.27.1.
23. Add flake-vs-go.mod version guard (fail fast on drift).
24. `nix build .` + `nix build .#visionreviewd` + `nix flake check` after the bump.
25. Re-run the full 9-step verification matrix (both JSON regimes, govulncheck, tidy-diff).
26. Confirm `GOEXPERIMENT=jsonv2` CI job + `no-jsonv2` SDK-subset job still green post-changes.
27. Delete the toolchain-workaround paragraph from AGENTS.md once the flake is right.
28. Sweep AGENTS.md for other claims invalidated by the 1.27 switch (e.g. "GO toolchain: go.mod and the nixpkgs lock have been on 1.26.7" bullet is now self-contradictory).

**Dedup-thread follow-ups:**
29. Add the direct `defaultIDs` table test (both empty / one empty / neither).
30. CHANGELOG entry covering rounds 1–5 of the dedup pass.
31. Add a clarifying comment at `Generate`'s caller-override fallback distinguishing it from `defaultIDs`' constant-fallback rule (anti-split-brain).
32. Replace the hardcoded `"main"` in `GenerateOptions.SurfaceID`'s doc comment with a reference to `defaultSurfaceID`.
33. Sweep for other doc comments hardcoding values that live in constants.
34. Add `// art-dupl:accept` consistency check: `buildObjectCall` (structured.go:199) shares the params-copy idiom but carries no marker — decide marker-for-consistency or document why not.
35. Make the accepted-groups table machine-checkable (script: re-run ladder, assert counts, assert markers exist at recorded sites).
36. Add the art-dupl ladder (`-t 3` = 0) to the pre-release checklist.
37. Re-verify DUPLICATION_POLICY claims via docs-health VERIFY after any future dedup pass.

**Docs-health / harvesting:**
38. HARVEST round-2 report (f) items into TODO_LIST/ROADMAP.
39. HARVEST this report's (f) items (same pass, one routing decision per item).
40. Annotate round-1 + round-2 reports as superseded-by-round-3 for current-state sections.
41. docs-health VERIFY pass over AGENTS.md dedup + toolchain sections against actual scanner/flake output.
42. Consider a `docs/DOMAIN_LANGUAGE.md` entry for the ID-fallback rule ("effective surface/catalog ID") now that it has one home.

**Trust-in-tooling:**
43. Restart gopls to clear the 4 phantom `newConfigFlagSet` errors; if they persist, capture and file against crush-config.
44. Document the "trust the CLI" verification recipe (build/vet/lint) next to the stale-LSP AGENTS.md bullet with this round's fresh example.
45. File/implement upstream or local: art-dupl output determinism (stable group ordering for diffable runs).
46. Record the `compareArgCount`-vs-literal scanner behavior in DUPLICATION_POLICY (type-aware mode is token-based, not semantic — named constants break grouping).

**Smaller hygiene noticed this round:**
47. `/tmp/vra-gocache` used this session — throwaway, fine per policy; ensure no durable evidence points at it.
48. Confirm the auto-commit daemon committed all round-3 files (surface.go, generate.go, policy doc, AGENTS.md, this report) and nothing is half-staged.
49. Example builds (`examples/a2ui`, `examples/structured`) under both JSON regimes after their marker edits from round 2 (never built since).
50. Tiny: `pkill`/ignore — `GOCACHE=/tmp/vra-gocache` reuse across commands saved minutes; consider documenting a stable throwaway-cache path in AGENTS.md as the sanctioned pattern (replacing per-command `mktemp -d`).

## g) Up to 3 questions I can NOT figure out myself

1. **Go 1.27.1 bump:** intentional and finished, or half-done? If finished: should I align the flake devShell + CI + AGENTS.md as one change (items 22–28)? If half-done: what is the target state?
2. **The uncommitted `.golangci.yaml`** (wsl_v5 / exhaustruct_v5 / gocognit / tagliatelle): is this yours/intentional? Should I fix the 11 findings under those rules, or revert the config? And if keep: was the CI lint agreement ever checked?
3. **a2ui wire output contract:** must wire output be byte-stable (deterministic key order → sort keys in marshal, re-pin goldens), or is semantically-equal output sufficient (then `ExampleCompile`'s expectation is over-strict and should be relaxed instead)?

_(Same 3 questions as round 2 — re-asked because each gates a distinct block in §f.)_

---

**Immediate next step when work resumes:** docs-health HARVEST over items 38–39, gated on your answers to §g.

_Waiting for instructions._
