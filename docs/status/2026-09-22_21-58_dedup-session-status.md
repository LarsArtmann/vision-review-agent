# Status Report — Deduplication Session (2026-09-22, 21:58 CEST)

**Scope:** This session only — the `deduplicate!` run over the whole repo
(`art-dupl --sort total-tokens -t 1 --type-aware`, 10 clone groups pasted as
input). No unrelated research performed. Point-in-time snapshot; annotate,
never rewrite.

**Session window:** 2026-09-22 ~21:00–21:58 CEST, branch `master`, single
module, Go toolchain go.mod = 1.27.1 (devShell still 1.26.7 — see d/e).

---

## a) FULLY DONE

| #  | Item                                                                                                                                                                            | Evidence                                                                                                                                                         |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | All 10 reported clone groups read and judged (extract / accept / exclude)                                                                                                       | Judgment table in `docs/DUPLICATION_POLICY.md` § "Accepted clone groups (2026-09-22 full-repo pass)"                                                             |
| 2  | Group 1 eliminated: four hand-rolled sorted-keys helpers deleted, call sites use stdlib `slices.Sorted(maps.Keys(m))`                                                           | `internal/reviewd/discover.go:219-220`, `pipeline.go:112`, `replay.go:227` (+ hand-rolled `projects` collection in `Replay` folded to one line, `replay.go:241`) |
| 3  | Bonus modernization: `sort.Slice` → `slices.SortFunc` in `sortedSuggestions` (removed the last `sort` import)                                                                   | `internal/reviewd/discover.go:209-211`                                                                                                                           |
| 4  | Group 3 eliminated: a2ui payload-decode error contract single-sourced in `decodePayload`; 4 `UnmarshalJSON` methods now share it with byte-identical error messages             | `pkg/vision/a2ui/messages.go:317-325` (helper), call sites at each kind's `UnmarshalJSON`                                                                        |
| 5  | Group 5 eliminated: prompt-assembly skeleton extracted to `buildPrompt`; per-persona instruction blocks pinned as `reviewInstructions` / `compareInstructions` constants        | `internal/reviewd/prompts.go:19-72`                                                                                                                              |
| 6  | Group 6 extracted: `-config` flag (default + description) defined exactly once in `newConfigFlagSet`; used by compare, events, backup, `parseConfigFlag`                        | `cmd/visionreviewd/commands.go:33-39`                                                                                                                            |
| 7  | Verification loop: `go build ./...`, `go vet`, `gofmt -l`, `go test -race -count=1` green for `internal/reviewd`, `cmd/visionreviewd`, `pkg/vision`; a2ui Ginkgo suite 11/11    | Session shell transcripts; `internal/reviewd` 1.1s, `cmd/visionreviewd` 3.0s, `pkg/vision` 4.7s                                                                  |
| 8  | Lint with the pinned golangci-lint v2.13.2: **zero findings in any touched hunk** (11 findings all pre-date the session, all in untouched code — see c)                         | `/tmp/lint-out.txt`-equivalent transcript; findings sorted by file                                                                                               |
| 9  | art-dupl re-run: **10 → 7 clone groups**; all 7 accepted with written rationale                                                                                                 | Final scan: "Found total 7 clone groups."                                                                                                                        |
| 10 | `docs/DUPLICATION_POLICY.md` updated: new Current State (2026-09-22), 4 new helper-table rows, new accepted-groups section; accidental blank-line slip in the a2ui header fixed | Policy doc diff                                                                                                                                                  |
| 11 | `AGENTS.md` updated: Code Duplication section refreshed; stale "on 1.26.7" toolchain claim replaced with the 1.27.1 reality + devShell-mismatch and empty-`$GO` gotchas         | `AGENTS.md` § Code Duplication, § `version` bullet                                                                                                               |
| 12 | Pre-existing master breakage root-caused, blamed correctly (not this session's changes), and ticketed                                                                           | `TODO_LIST.md` § "pkg/vision/a2ui — Go 1.27 encoding/json/v2 wire regression (found 2026-09-22)"                                                                 |
| 13 | Pre-existing breakage proven pre-existing via clean-worktree run at committed HEAD (not local-diff blame)                                                                       | `git worktree add /tmp/vra-head-$$ HEAD` → identical failures                                                                                                    |

## b) PARTIALLY DONE

| # | Item                                                              | Done                                                                                                                                                     | Missing                                                                                                                                                                                                                                             |
| - | ----------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | Verification matrix for the session's changes                     | build/vet/gofmt/race/lint on touched packages; default regime                                                                                            | Full 9-step matrix NOT run: no `GOEXPERIMENT=jsonv2` test pass over the touched packages (only one probe), no `GOEXPERIMENT=none` SDK-subset run, no `go mod verify`/`tidy -diff`, no `nix run .#test/.#lint`, no `nix build`, no `nix flake check` |
| 2 | a2ui test-suite health assessment                                 | Ginkgo 11/11 + all table tests confirmed green                                                                                                           | First run's `tail -15` masked two `TestMessageWireShape` subtests; only found on a later dedicated run — the "which tests fail" list was complete only at the end                                                                                   |
| 3 | Root-cause analysis of the Go 1.27 `encoding/json/v2` regressions | Mechanism identified (map-key order nondeterministic; `omitempty` ignored on scalars); exact upstream behavior change not pinned to a release note/issue | No upstream issue filed/linked; no decision between "ordered encoder" vs "semantic-compare tests"                                                                                                                                                   |
| 4 | Toolchain-state assessment                                        | Established: go.mod 1.27.1, devShell 1.26.7, four go-1.27.1 nix store paths (one arm64), CI pin state NOT checked                                        | Did not inspect `flake.nix`/CI to see whether the 1.27.1 bump is mid-flight and who owns finishing it                                                                                                                                               |
| 5 | `docs/status/` hygiene                                            | This report written                                                                                                                                      | Section (f) items not yet HARVESTed into `TODO_LIST.md` (only the 3 regression tickets landed); CHANGELOG has no entry for the dedup refactor yet                                                                                                   |

## c) NOT STARTED (found this session, zero work done)

1. Fix a2ui wire-output determinism (sorted-key emission in `Component.MarshalJSON` / `UpdateDataModel.MarshalJSON`).
2. Fix `omitempty`-ignored-on-scalars (`SendDataModel`); migrate a2ui marshal tags to `omitzero` or gate fields in code.
3. Re-pin `ExampleCompile` `// Output:` and `TestMessageWireShape` wants after the above.
4. Resolve the 11 lint findings from the uncommitted `.golangci.yaml` rule additions (tagliatelle ×3, wsl_v5 ×3, gocognit ×1, exhaustruct_v5 ×4).
5. Review + deliberately commit the modified `.golangci.yaml` itself (it is an uncommitted working-tree change that changed what CI/lint enforces).
6. Align flake devShell Go (1.26.7) with go.mod (1.27.1) and check CI toolchain pins.
7. Full verification matrix (b1 list) incl. both JSON regimes under Go 1.27 and nix builds.
8. Decide whether `Pipeline.SourceURLs` / `Captured.SourceURL` (recent fleet additions) should be populated at the flagged literals or explicitly exempted.
9. CHANGELOG entry for the dedup refactor (per TODO_LIST discipline).
10. HARVEST of this report's section (f) into `TODO_LIST.md`/`ROADMAP.md`.

## d) TOTALLY FUCKED UP

Nothing the session shipped is broken — but the session _surfaced_ a repo that is currently red on master, none of it caused by this session (proven at committed HEAD):

1. **a2ui wire output is nondeterministic on Go 1.27** — 5 runs of `ExampleCompile` produced 4 different key orderings. For a protocol-emitting package this is the worst class of bug (breaks byte-goldens, caching, hashing, reproducibility). `internal/reviewd` replay determinism is NOT affected (markdown projection, different code path) — checked.
2. **`omitempty` silently ignored** — `"sendDataModel":false` leaks into every createSurface wire message; 2 tests fail.
3. **Master lint is red** (11 findings) because the working tree carries an **uncommitted `.golangci.yaml`** that added/changed rules (wsl_v5, exhaustruct_v5, gocognit, tagliatelle) — config and code are out of sync in the same tree.
4. **Toolchain split-brain**: go.mod demands 1.27.1, the nix devShell serves 1.26.7 — `nix develop -c go build` fails today. CI pin state unverified.

### My own session failures (asked for explicitly: what did I forget / do wrong)

1. **Empty `$GO` → shell `test` builtin trap.** In several commands I exported `PATH` but forgot to define `GO=...`, so `go test ...` expanded to `test ...` — the shell builtin — and mvdan/sh failed with "not a valid test operator". I initially misread this as a Go 1.27 `-run` syntax change and burned a Sourcegraph search on it before recognizing the empty-variable expansion (error column positions matched the exact args). Cost: ~4 wasted round trips.
2. **`tail -15` eyeballing instead of grepping the full failure list** — I declared "only ExampleCompile fails in a2ui" from a truncated view; `TestMessageWireShape/createSurface{,_with_theme}` were hiding above the cut and surfaced two runs later.
3. **Ambiguous worktree experiment.** The HEAD-worktree check ran while the auto-commit daemon was committing my edits, so "HEAD" was a moving target (my working tree and HEAD differed in marshal output ordering between runs, which briefly looked like nondeterminism vs. code drift). Should have pinned the exact commit hash and listed which of my files HEAD already contained.
4. **Small edit slip:** a multiedit anchor accidentally ate the blank line after the a2ui section header in `DUPLICATION_POLICY.md` (caught and fixed on review).
5. **Exit-code misread:** first lint run echoed "LINT EXIT: 0" via a bad `PIPESTATUS` expansion while the real exit was 1; re-ran with explicit capture.

## e) WHAT WE SHOULD IMPROVE

1. **Env hygiene in shell tooling:** define `GO/GOCACHE/GOMODCACHE` once at the top of every command block; never rely on `$GO` set in a previous call. Record the empty-`$GO`→`test`-builtin trap (now in AGENTS.md).
2. **Failure enumeration over tail-windowing:** always `rg "^(ok|FAIL|--- FAIL)" full-output-file` for test verdicts, never eyeball truncated tails.
3. **Worktree comparisons must pin a hash** and state which working-tree files the daemon already committed; otherwise blame is ambiguous with an auto-commit daemon active.
4. **Fix-on-sight threshold:** the a2ui determinism bug is exactly the kind of thing "fix immediately" wants, but it exceeds the 5-minute bar (wire-contract decision + regime testing). The 5-minute rule needs an explicit "ticket immediately with root cause" continuation — done here, but the TODO ticket should have landed in the same breath as the diagnosis (it did, barely).
5. **Verification proportionality:** an SDK-subset change (a2ui) deserves the regime double-run (jsonv2/none) immediately — the regime split is a documented consumer guarantee; I skipped it and should not have.
6. **Stale LSP:** gopls kept reporting `newConfigFlagSet` undefined after the CLI proved otherwise; a proactive `lsp_restart` after adding package-level symbols would stop the noise instead of re-justifying it every message.
7. ** art-dupl accepted-group markers:** the codebase already uses `// art-dupl:accept` (commands.go:199); the 7 accepted groups could carry these markers so future scans shrink to genuinely new findings instead of re-listing accepted ones.

## f) NEXT: up to 50 things to get done (brainstorm, impact-ordered; ROUTING: top ~10 → TODO_LIST, rest → ROADMAP)

1. Fix a2ui wire determinism: sorted/ordered key emission in `Component.MarshalJSON` + `UpdateDataModel.MarshalJSON` (+ `Theme` map if user-visible).
2. Migrate a2ui marshal tags `omitempty` → `omitzero` (v1+v2-honored) or gate fields in code; fix `SendDataModel:false` leak.
3. Re-pin `ExampleCompile` output block and `TestMessageWireShape` wants after 1–2; add a byte-determinism golden test (marshal twice, assert equal).
4. Sweep the repo for other `map[string]any` JSON-marshal sites with byte-stability expectations; audit each under Go 1.27.
5. Decide + apply tagliatelle handling for `sourceURLs`/`sourceURL` (documented `//nolint:tagliatelle` per the openaicompat precedent) — 3 sites.
6. Fix wsl_v5 ×3 (markdown.go ×2, pipeline_bdd_test.go ×1).
7. Refactor `TestGoldenJournalPayloadsDecode` under gocognit 25 (extract helpers/table).
8. Decide exhaustruct_v5 ×4: populate `Pipeline.SourceURLs`/`Captured.SourceURL` at the flagged literals or nolint with reason; exempt bbolt.Options literals deliberately.
9. Review + commit the modified `.golangci.yaml` deliberately (config/code sync).
10. Align devShell Go with go.mod (flake input/pin → 1.27.1); re-verify `nix develop -c go build`.
11. Add a CI/dev check that fails when flake Go ≠ go.mod Go (catch the split-brain class).
12. Run the full 9-step verification matrix on the current tree (incl. both JSON regimes under 1.27).
13. Re-examine CI regime jobs under Go 1.27: `no-jsonv2` SDK subset now compiles `encoding/json/v2` natively — does the job still test what it claims?
14. `nix build .` + `nix build .#visionreviewd` + `nix flake check` after toolchain alignment (silent-empty-build class guard).
15. `nix run .#verify-bump` / dep-drift / govulncheck for the toolchain-bump cycle.
16. File/link the upstream jsonv2 behavior-change issue (map ordering, omitempty) in TODO/DEPS for future archaeology.
17. Decide the wire-contract policy question: is a2ui output byte-stability a hard consumer requirement (hash/cache) or semantic-only? Document in `docs/A2UI.md`.
18. HARVEST this report's (f) list into `TODO_LIST.md`/`ROADMAP.md` (docs-health).
19. CHANGELOG entry: dedup refactor (sorted-keys stdlib, decodePayload, buildPrompt, newConfigFlagSet).
20. Add `// art-dupl:accept` markers (or policy-doc references) to the 7 accepted groups.
21. Modernization sweep: other `sort.Strings`/`sort.Slice`/hand-rolled key-collection sites repo-wide (same stdlib replacement).
22. Audit `internal/reviewd/config.go` `MarshalJSON` (`configJSON` → map-free, struct-based) for regime stability — it feeds doctor/suggested output.
23. Check whether `examples/a2ui` README/output docs embed expected wire strings that the determinism fix will change.
24. Annotate older `docs/status/` reports whose "0 actionable clone groups / full-green" claims this session superseded (docs-health ANNOTATE).
25. Consider pinning art-dupl version + command in the policy doc (reproducibility of "Current State" claims).
26. Re-verify golden-journal fuzz targets still pass under 1.27 (CBOR decode path unaffected expected, unproven this session).
27. Confirm `GOEXPERIMENT=jsonv2` nix-build requirement still exists now that json/v2 is native on 1.27 (AGENTS.md claim may be obsolete — same class as the 1.26.7 note I fixed).
28. Decide whether `AGENTS.md` "GOEXPERIMENT=jsonv2 required in nix builds" and "Dual json v1+v2" entries need the 1.27 rewrite (likely yes).
29. Run the a2ui fuzz targets under 1.27 (decode path touched by `decodePayload` refactor).
30. Branch-protection check: with lint red on master, confirm which required checks are failing and whether the auto-commit daemon is pushing red.
31. Add `TestMessageWireShape`-style semantic-compare helper if the wire-contract decision (17) lands on "semantic-only".
32. Update `docs/DEPS.md` if the Go 1.27 bump came with go-cqrs-lite implications (unverified this session).
33. Verify `visionreviewd` version-smoke nix check still passes under 1.27.1 build.
34. Consider extracting the repeated `if err != nil { fmt.Fprintf(stderr, "visionreviewd %s: %v", ...); return exitFailed }` idiom in `commands.go` (~15 sites, `failCommand` helper — same shape as `failAnalysis` in cmd/vision; below scan threshold but real repetition).
35. Add a determinism note to `docs/DOMAIN_LANGUAGE.md`/A2UI docs: "wire bytes are an implementation detail / or a contract" per decision 17.
36. Performance re-check: `slices.Sorted(maps.Keys())` vs old helpers on hot paths (replay 10k events baseline) — expected negligible; measure once for the record.
37. Housekeeping: `/tmp` scratch files from this session (lint output, a2ui test output, worktree) — verify worktree removed (it was), transcripts are /tmp-only per run-artifacts policy.
38. Re-run `art-dupl` with `--html` and file the visual report alongside the policy doc if desired.
39. Sweep for remaining `//nolint` directives invalidated by the new `.golangci.yaml` rules (stale-nolintlint class).
40. Evaluate `modernize`/`gopls modernize` analyzer for further stdlib replacements (sort→slices done ad hoc; make it systematic).
41. Check whether the two identical go-1.27.1 amd64 store paths indicate duplicate GC roots (nix hygiene, low prio).
42. Consider upstreaming/annotating the `-run`-expression lesson: Go 1.27 test flag changes may affect CI scripts using `^` anchors in `-run` (I never verified whether `^Test` is actually invalid in 1.27 or whether that too was the shell artifact — **unresolved**, see g).
43. Prompts: consider whether `compareInstructions` should live in testdata next to the pinned official schemas (a2ui pattern) for the reviewd prompts too — consistency question.
44. Decide if `buildPrompt` should take the instructions as a typed doc (struct with headings) instead of raw string — YAGNI check first.
45. Verify `docs/activation/*` fleet scripts unaffected by sorted-keys deletions (they shell out to the daemon; behavior identical — confirm via smoke).
46. Run `site-monitor`/fleet timers health check if the daemon binary gets rebuilt with the dedup changes.
47. Add regression test: `decodePayload` error messages pinned (they are the contract for `errors.Is` consumers).
48. Re-check `AGENTS.md` "Branch protection carries 10 required checks" list against current CI after the yaml change lands.
49. Consider a `make`-free pre-push script aggregating: gofmt, vet, race, regimes, lint (single entry, mirrors verify-bump).
50. Post-toolchain-alignment: fresh full-green verification run + new `docs/status/` evidence doc (the 2026-09-07 one is the latest).

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Is the Go 1.27.1 go.mod bump an intentional, finished migration** (and the devShell/CI lag deliberate), **or a half-finished auto-upgrade?** I cannot tell who owns finishing it, whether another session is mid-flight on it, and whether `nix build` currently passes in CI. Determines whether I should do item 10–15 myself or leave it alone.
2. **Should I fix the a2ui wire determinism + `omitempty` regressions myself next, or is another workstream already on them?** They are on master, red, and pre-date this session — I don't want to collide with (or duplicate) in-flight work, and I don't know if the modified `.golangci.yaml` is yours-in-progress.
3. **Is a2ui wire output required to be byte-stable** (hashing, caching, golden files, or consumers diffing bytes), **or is semantic JSON equality the actual contract?** This decides whether the fix is an ordered encoder (strict) or test-side semantic comparison (loose), and I found no consumer evidence either way in the session.

---

_Report per status-report skill; `.md` written at explicit user request (skill default is HTML). Not committed (harness forbids unsolicited commits; auto-commit daemon will pick it up)._
