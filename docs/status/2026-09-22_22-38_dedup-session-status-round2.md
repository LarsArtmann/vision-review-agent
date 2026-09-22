# Status Report — Deduplication Session, Round 2 (2026-09-22, 22:38 CEST)

**Scope:** This session only. Supersedes
`docs/status/2026-09-22_21-58_dedup-session-status.md` (round 1 report);
that file stays as written — this one extends it with the "not even these?"
follow-up round (21:58 → 22:38 CEST). Point-in-time snapshot; annotate,
never rewrite.

**Session thread so far:**
1. `deduplicate!` — art-dupl `-t 1 --type-aware`, 10 clone groups → judged,
   3+1 harmful eliminated, 7 accepted (round 1, ~21:00–21:58).
2. Round-1 status report written (`...21-58_dedup-session-status.md`).
3. User: "not even these?" (the 2 groups left at `-t 3`) → researched fantasy
   types, encoded verdicts with `// art-dupl:accept` markers → **`-t 3` scan
   now empty; `-t 1` down to 5 idiom-only groups** (22:00–22:38).

---

## a) FULLY DONE

| # | Item | Evidence |
| - | ---- | -------- |
| 1 | Round-1 dedup (all of it; see 21:58 report §a): 3 harmful groups eliminated via stdlib/`decodePayload`/`buildPrompt`, plus `newConfigFlagSet`, `slices.SortFunc` modernization; docs updated | Round-1 report; `docs/DUPLICATION_POLICY.md` |
| 2 | Round-1 status report written at user-requested `.md` (skill's HTML default overridden by explicit user instruction — flagged in file footer) | `docs/status/2026-09-22_21-58_dedup-session-status.md` |
| 3 | **Round 2 — fantasy evidence gathered before final verdict:** read `charm.land/fantasy@v0.45.0/agent.go` — `AgentCall` (:164) and `AgentStreamCall` (:273) are flat, non-embedded, non-convertible duplicate structs; Go generics have no field constraints → the 6-line pointer copy is the API boundary, all merge paths (reflection / wrapper interfaces / tuple assignment / codegen) evaluated and rejected on type-safety or net-code grounds | Module-cache transcript in session; rationale now in policy doc |
| 4 | **Round 2 — verdicts encoded at the sites** with `// art-dupl:accept` markers (the codebase's existing convention, commands.go:199): vision.go both builders, examples/a2ui + examples/structured mains | `pkg/vision/vision.go` (both `build*Call`), both example `main.go`s |
| 5 | **Round 2 — scanner verified empirically:** `art-dupl -t 3` → **0 clone groups**; `-t 1` → 5 (down from 7; remaining 5 are pure Go-idiom noise: unrelated `if err != nil` pairs ×2, two-line defaulting pair, `newConfigFlagSet` call lines, `NArg` usage checks) | Session scan transcripts |
| 6 | Round 2 — build/gofmt/tests still green after markers: `go build ./...`, `gofmt -l` clean, `pkg/vision` race suite ok | Session transcript |
| 7 | Policy doc updated with round-2 evidence: fantasy agent.go line refs in the accepted-groups table, marker locations, and the `-t 3` empty-scan state | `docs/DUPLICATION_POLICY.md` § Accepted clone groups |
| 8 | Pre-existing master breakage root-caused, blamed correctly via clean-worktree run at committed HEAD, ticketed (from round 1, still standing) | `TODO_LIST.md` § a2ui Go 1.27 regression |

## b) PARTIALLY DONE

| # | Item | Done | Missing |
| - | ---- | ---- | ------- |
| 1 | Verification matrix for session changes | build/vet/gofmt/race/lint (pinned v2.13.2) on touched packages, default regime | No `GOEXPERIMENT=jsonv2` / `GOEXPERIMENT=none` regime runs over the touched SDK packages, no `go mod verify`/`tidy -diff`, no `nix run .#test/.#lint`, no `nix build`, no `nix flake check` |
| 2 | Duplication policy end-state | `-t 3` empty; every remaining `-t 1` group documented with rationale | The 5 remaining `-t 1` groups carry rationale only in the policy doc, no in-place markers (deliberate — marking Go idioms is scanner-appeasement; revisit if you disagree) |
| 3 | a2ui Go 1.27 regression triage | Root cause + mechanism + blame isolated; 3 tickets filed | The actual fixes (deterministic marshal, `omitempty`→`omitzero`, re-pinned wants) not started; upstream issue not filed |
| 4 | Toolchain-state assessment | go.mod 1.27.1 vs devShell 1.26.7 confirmed + workaround documented in AGENTS.md; 4 nix store paths for go-1.27.1 (one arm64) | flake/CI pin ownership and the uncommitted `.golangci.yaml` change not investigated (out of session scope) |
| 5 | Answers to round-1 report's 3 questions (g) | Asked | **Not answered** — re-asked below (g) |

## c) NOT STARTED (all found by this session, zero work done)

1. Fix a2ui wire-output determinism (sorted-key emission in `Component.MarshalJSON` / `UpdateDataModel.MarshalJSON`).
2. Fix `omitempty`-ignored-on-scalars (`SendDataModel:false` leak); migrate a2ui marshal tags to `omitzero` or gate in code.
3. Re-pin `ExampleCompile` `// Output:` and `TestMessageWireShape` wants after 1–2.
4. Resolve 11 pre-existing lint findings (tagliatelle ×3, wsl_v5 ×3, gocognit ×1, exhaustruct_v5 ×4) from the uncommitted `.golangci.yaml` rule additions.
5. Review + deliberately commit the modified `.golangci.yaml` itself.
6. Align flake devShell Go (1.26.7) with go.mod (1.27.1); check CI pins.
7. Full verification matrix incl. both JSON regimes under Go 1.27 + nix builds.
8. Decide populate-vs-nolint for `Pipeline.SourceURLs` / `Captured.SourceURL` literals.
9. CHANGELOG entry for the dedup refactor.
10. HARVEST section (f) below into `TODO_LIST.md`/`ROADMAP.md` (docs-health).

## d) TOTALLY FUCKED UP

Nothing this session shipped is broken. The repo state this session *found* (unchanged since round 1, all pre-existing, proven at committed HEAD):

1. **a2ui wire output nondeterministic on Go 1.27** (5 runs → 4 orderings) — worst-class for a protocol emitter; 2 tests fail (`ExampleCompile`, `TestMessageWireShape/createSurface{,_with_theme}`).
2. **`omitempty` ignored on scalars** under native `encoding/json/v2` → `"sendDataModel":false` leaks into every createSurface message.
3. **Master lint red** (11 findings) driven by an **uncommitted `.golangci.yaml`** — config and code out of sync in one tree.
4. **Toolchain split-brain** — go.mod 1.27.1 vs devShell 1.26.7; `nix develop -c go build` fails today.

### My own session failures (asked for explicitly)

1. **Round 1: accepted the vision.go group without researching fantasy's actual type layout.** I ruled from general Go knowledge ("external types, no interface") — correct conclusion, but unverified. The user's "not even these?" forced the fantasy source read in round 2; the evidence (flat structs, agent.go:164/273) should have been gathered in round 1. This is exactly the verify-external-claims discipline I apply to claims but under-applied to refactoring verdicts.
2. **Empty `$GO` → shell `test` builtin trap** (round 1): forgot `GO=` in some commands; `go test` became `test`, mvdan/sh errored "not a valid test operator"; I first misdiagnosed it as a Go 1.27 `-run` syntax change and spent a Sourcegraph search before seeing the empty expansion. ~4 wasted round trips.
3. **`tail -15` eyeballing** (round 1): declared "only ExampleCompile fails" from a truncated view; `TestMessageWireShape` subtests surfaced two runs later. Always grep the full verdict list.
4. **Ambiguous worktree experiment** (round 1): HEAD-worktree check raced the auto-commit daemon; "HEAD" was a moving target between runs. Should have pinned the hash and listed which files HEAD already contained.
5. **Round 2: first marker edit failed** — old_string appeared in both identical builders and I passed `replace_all` implicitly false. I *knew* the block was duplicated (that was the finding); should have anticipated non-uniqueness. Fixed immediately with `replace_all: true` (which was the desired outcome anyway).
6. **Minor:** bad `PIPESTATUS` echo reported "LINT EXIT: 0" while the real exit was 1 (round 1); a multiedit anchor ate a blank line in the policy doc (round 1, caught and fixed).

## e) WHAT WE SHOULD IMPROVE

1. **Research externals before encoding refactoring verdicts** — the "accept" that survives a challenge is the one with sources cited (fantasy agent.go:164/273). Round 2's verdict is durable; round 1's was luck.
2. **Encode accept-decisions as `// art-dupl:accept` at the site** by default, policy doc as the index — future scans stay clean and the rationale travels with the code (now done for the 2 judgment groups).
3. **Shell env hygiene:** define `GO/GOCACHE/GOMODCACHE` at the top of every command; never depend on prior calls (the empty-`$GO` trap is now in AGENTS.md).
4. **Test verdicts by grep, never by tail** — one `rg "^(ok|FAIL|--- FAIL)"` over full output.
5. **Worktree comparisons must pin the commit hash** when an auto-commit daemon is live.
6. **SDK-touching changes deserve both JSON-regime runs immediately** — the dual-regime guarantee is documented; I keep deferring it.
7. **Restart the LSP after adding package-level symbols** instead of re-justifying stale "undefined" errors every message (gopls still shows `newConfigFlagSet` undefined; CLI proved clean long ago).
8. **When challenged ("not even these?"), re-research instead of re-defending** — it produced a better end-state (-t 3 → 0) at near-zero cost.

## f) NEXT: up to 50 things (brainstorm, impact-ordered; top ~10 → TODO_LIST, rest → ROADMAP)

1. Fix a2ui determinism: ordered key emission in `Component.MarshalJSON` + `UpdateDataModel.MarshalJSON` (+ `Theme` map).
2. `omitempty` → `omitzero` migration (or code-gated fields) across a2ui marshal tags.
3. Re-pin `ExampleCompile` output + `TestMessageWireShape` wants; add a marshal-twice byte-equality golden test.
4. Resolve 11 lint findings: tagliatelle `sourceURLs`/`sourceURL` nolints ×3 (openaicompat precedent), wsl_v5 ×3, gocognit refactor of `TestGoldenJournalPayloadsDecode`, exhaustruct_v5 ×4 (populate or justified nolint; bbolt.Options literals deliberate).
5. Review + commit the modified `.golangci.yaml` deliberately (config/code sync).
6. Align devShell Go with go.mod (1.27.1); add a flake-vs-go.mod version guard so the split-brain class cannot recur.
7. Full 9-step verification matrix on the current tree, both JSON regimes under 1.27.
8. Re-examine CI `no-jsonv2` job semantics on 1.27 (SDK subset now compiles json/v2 natively — does the job still test what it claims?).
9. `nix build .` + `.#visionreviewd` + `nix flake check` after alignment (silent-empty-build guard).
10. Decide the a2ui wire contract: byte-stability (hash/cache/goldens) vs semantic-only — document in `docs/A2UI.md`; determines fix strictness for items 1–3.
11. HARVEST this (f) list into TODO_LIST/ROADMAP (docs-health).
12. CHANGELOG entry for the dedup refactor (round 1 + round 2 markers).
13. Answer/resolve the 3 open questions in (g) — several items above branch on them.
14. File/link upstream jsonv2 behavior-change issue (map ordering + omitempty) in TODO/DEPS.
15. Sweep repo for remaining `sort.Strings`/`sort.Slice`/hand-rolled key collection → `slices`/`maps`.
16. Sweep for other `map[string]any` JSON-marshal sites with byte-stability expectations (e.g. `configJSON` marshal feeding doctor output).
17. Decide populate-vs-nolint for `Pipeline.SourceURLs` / `Captured.SourceURL` (exhaustruct).
18. Verify stale `//nolint` directives under the new lint config (nolintlint class).
19. Re-verify golden-journal fuzz targets under 1.27 (CBOR path; `decodePayload` refactor touched decode error wrapping).
20. Check whether `GOEXPERIMENT=jsonv2` nix-build requirement (AGENTS.md) is obsolete now that json/v2 is native — same class as the 1.26.7 note I fixed.
21. Rewrite AGENTS.md "Dual json v1+v2" entry for the 1.27 reality if confirmed.
22. Branch-protection reality check: which required checks are red on master right now; is the auto-commit daemon pushing red?
23. `examples/a2ui` docs/README may embed expected wire strings — update after determinism fix.
24. Annotate superseded status reports claiming "0 actionable clone groups / full-green" (docs-health ANNOTATE).
25. Pin art-dupl version + canonical command in DUPLICATION_POLICY.md (reproducible "Current State" claims).
26. Consider `failCommand(stderr, name, err) int` extraction in commands.go (~15 sites of the 4-line failure idiom; same shape as `failAnalysis`).
27. Performance note: `slices.Sorted(maps.Keys())` vs old helpers on replay hot path — measure once for the record.
28. Confirm worktree `/tmp/vra-head-*` cleanup (it was removed) and keep /tmp transcripts uncited per run-artifacts policy.
29. Consider a2ui prompt-instruction constants living beside pinned official schemas (consistency with a2ui pin-test pattern) — YAGNI check first.
30. Add pinned test for `decodePayload` error messages (they are the `errors.Is` contract).
31. Evaluate `gopls modernize` analyzer run repo-wide for further stdlib replacements.
32. Verify `visionreviewd` version-smoke nix check under the 1.27.1 build.
33. Re-check branch-protection required-checks list vs current CI after the yaml lands.
34. Consider aggregate pre-push gate (gofmt+vet+race+regimes+lint) mirroring verify-bump.
35. Full-green verification run + fresh evidence status doc after the toolchain lands (latest is 2026-09-07).
36. `docs/DEPS.md`: check go-cqrs-lite implications of the 1.27 bump (unverified).
37. Nix hygiene: why two identical amd64 go-1.27.1 store paths (duplicate GC roots?).
38. Fleet smoke: rebuild daemon with dedup changes, confirm site-monitor/fleet timers unaffected.
39. ROADMAP: a2ui v1.0 spec migration (pre-existing) — untouched, still open.
40. ROADMAP: consider upstreaming `art-dupl:accept` marker documentation/example to art-dupl README (tool-level contribution).
41. ROADMAP: structured "wire determinism" lint/test helper package if more packages adopt byte-stable output contracts.
42. ROADMAP: explore `jsontext` ordered-encoder utilities vs hand-rolled sorted emission for the a2ui fix.
43. TODO_LIST already carries 3 regression tickets — mark their provenance (session, HEAD hash) for future annotation.
44. Consider committing the two status reports (21:58 + this) — auto-daemon will, but explicit message would be cleaner.
45. Re-run `art-dupl -t 1` after each future PR and record the count in the policy doc (living metric).
46. Evaluate whether the 5 remaining `-t 1` groups deserve in-place markers after all (policy decision, owner's call).
47. Check `internal/reviewd` config marshal path under json v1 vs v2 for output stability (feeds `discover`/`doctor` output users may diff).
48. Sweep examples for other shared idioms worth `examples/internal` promotion — only if ≥3 examples share non-trivial logic (bar: teaching-code self-containment).
49. Verify fuzz seeds/corpus under 1.27 for `pkg/vision` image paths (round-1 changes touched decode error wrapping there too? — no, only a2ui; confirm and close).
50. Post-alignment: re-read AGENTS.md top-to-bottom for other stale claims surfaced by the 1.27 bump (the 1.26.7 note was one; there may be siblings).

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF (re-asked — round-1 report's (g) went unanswered)

1. **Is the Go 1.27.1 go.mod bump an intentional, finished migration (devShell/CI lag deliberate), or a half-finished auto-upgrade?** Decides whether items 6–9 are mine to execute or another workstream's.
2. **Should I fix the a2ui determinism + `omitempty` regressions next, or is in-flight work on them already?** Master is red; I don't want to collide with (or duplicate) yours or another session's changes — especially around the uncommitted `.golangci.yaml`.
3. **Is a2ui wire output required to be byte-stable** (hashing/caching/goldens/consumers diffing bytes), **or is semantic JSON equality the contract?** Decides whether the fix is an ordered encoder (strict) or test-side semantic comparison (loose); I found no consumer evidence either way in this session.

---

*Round-2 report per status-report skill; `.md` at explicit user request (skill default HTML). Not committed (harness forbids unsolicited commits; auto-commit daemon picks it up).*
