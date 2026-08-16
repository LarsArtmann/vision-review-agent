# visionreviewd Status — T6–T10 Complete (Pipeline, CLI, Daemon)

**Date:** 2026-08-16 21:18
**Session scope:** Fixed and finished T6, delivered T7–T10 of
[`docs/planning/2026-08-16_20-00_visionreviewd-full-execution-plan-v2.md`](../planning/2026-08-16_20-00_visionreviewd-full-execution-plan-v2.md).
T11–T18 remain. All work committed locally; nothing pushed.

## a) FULLY DONE (verified green at commit: build + tests + lint)

| Task   | Commit    | Content                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| ------ | --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| T6 fix | `b0d3bd5` | Score-trend test expectation (`PrevScore=ScoreUnknown` after first review), `initialViewState` var→function (gochecknoglobals), close-error join via `errors.Join` (errorlint), long lines (golines); go.mod gains go-cqrs-lite direct deps                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| T7     | `f9e77dd` | `scan.go`: globs→`Capture{ViewKey,Path,SHA256,ModifiedAt}`, newest-file-wins dedupe per view key, image-extension filter (reuses `isScreenshotExtension`), directories skipped. `pipeline.go`: `Pass` orchestration — skip-seen hashes → blob store → `RecordCapture` → auto-compare predecessor → `Review` → `RecordReview` → view markdown → INDEX refresh; per-view errors joined, one broken view never blocks the rest; model failure still leaves capture recorded, INDEX shows `?`. Ginkgo BDD suite (`reviewed_suite_test.go`, `helpers_test.go`, `pipeline_bdd_test.go`): new / unchanged / changed (asserts compare prompt carries 2 images, `▲ +1` trend) / model-fails |
| T8     | `4e8316d` | `compare.go`: `CompareManually(ctx, project, beforePath, afterPath)` — archives both blobs, view key derived from AFTER filename, records `view.compared`, writes comparison markdown, no synthetic captures. White-box tests incl. unparseable-name error path                                                                                                                                                                                                                                                                                                                                                                                                                    |
| T9     | `660e9d3` | `cmd/visionreviewd/` main.go + commands.go: 7 subcommands (run/once/discover/compare/events/replay/doctor) + version/help; `run(args, stdout, stderr) int` returns exit codes (0 ok / 1 fail / 2 usage), no `os.Exit` in parse; once/discover/compare fully wired; `reviewed.ExpandTilde` exported so CLI and config share one `~` rule; 11 table tests incl. discover E2E through the CLI                                                                                                                                                                                                                                                                                         |
| T10    | `de7c4c6` | `internal/reviewd/daemon.go`: `PassRunner` interface (Pipeline implements; tests wrap), `NewDaemon` validates via sentinels `ErrNoPipeline/ErrInvalidProjects/ErrInvalidInterval`, `Run` = immediate pass + ticker loop, pass failures logged-and-continue, returns nil on `context.Canceled`; CLI `run` wired with `signal.NotifyContext(SIGINT, SIGTERM)`; BDD spec proves ≥2 passes then clean nil shutdown. **Committed with 3 lint findings outstanding — see c)**                                                                                                                                                                                                            |

Also: `.golangci.yaml` — `misspell.ignore-rules: [reviewd]` in linters
settings (v2.12 schema key is `ignore-rules`; `ignore-words` is rejected).

Earlier-session commits for context: T2 `df51b84`, T3 `b9d873a`,
T4 `681915e`+`47a4137`, T5 `b24561a`.

## b) PARTIALLY DONE

- **T10 daemon** — code complete, tests + `-race` green, but committed with
  3 lint findings in `internal/reviewd/daemon_bdd_test.go` (gci import
  grouping; wrapcheck on `c.inner.Pass` return; wsl_v5 blank line in
  `startDaemon`). Needs a small fix commit.
- **CLI stubs** — `events`, `replay`, `doctor` subcommands exist, dispatch
  correctly, and fail with explicit `errNotImplemented`; bodies land in
  T11/T13.

## c) NOT STARTED

- **T11** events command (filters, pretty print) + replay command (rebuild
  reviews dir from journal) + replay test (wipe → regenerate → diff)
- **T12** E2E against fake OpenAI-compatible `/v1/chat/completions` server
- **T13** doctor command (config, dirs, globs, `/v1/models` model check)
- **T14** flake.nix `visionreviewd` buildGoModule package + stale vendorHash
- **T15** NixOS module (options, unit, optional llama-server on 8390)
- **T16** SystemNix lazy wrapper + activation docs
- **T17** README/CHANGELOG/AGENTS/TODO_LIST/FEATURES
- **T18** final full verification (`go test -race ./...`, vet, lint, gofmt)

## d) TOTALLY FUCKED UP (honest list)

1. **Missed the user's explicit instruction to write this status file** on
   first delivery — reported in chat only. Fixed now.
2. **Auto-git daemon committed T6 and T10 in RED/unpolished states** mid-task
   (T6 had 1 failing test + 3 lint findings; T10 has 3 lint findings). T6 was
   repaired with a follow-up commit; T10 still needs it. Lesson: commit (or
   let the daemon sweep) only after lint is zero, or expect a fixup commit.
3. **Wasted a cycle on `.golangci.yaml` misspell key**: tried `ignore-words`
   (rejected by the v2.12 schema), verified against the embedded
   `golangci.jsonschema.json` that the correct key is `ignore-rules`.
4. **First scan.go draft ingested non-image files** from bare `*` globs —
   caught by the test I wrote (`TestScanProjectIgnoresDirectoriesAndOtherFiles`
   failed 2≠1), fixed with the extension filter. Test-first paid off.
5. **`write` tool created `CompareManually` with hand-rolled
   `contains`/`indexOf` string helpers** before I caught it — replaced with
   `strings.Contains` in the rewrite. Never re-implement stdlib.
6. **First daemon BDD draft** referenced undefined helpers
   (`recordingPipeline`, `daemonTicksMu`) and a concrete `*Pipeline` field —
   the interface redesign (`PassRunner` + `countingRunner`) fixed both the
   design and the test.
7. **First daemon BDD AfterEach drained `stopped` twice** — the It's
   `Eventually(stopped)` consumed the value; AfterEach then timed out 5s.
   Fixed with a non-blocking `select` drain.

## e) WHAT WE SHOULD IMPROVE

- **Commit cadence vs daemon:** verify lint=0 BEFORE the auto-git daemon can
  sweep; otherwise always follow with a fixup commit (T10 pending).
- **wrapcheck in `_test.go`** bit us twice — internal helper wrappers must
  `%w`-wrap even in specs.
- **nolint placement:** must sit on its own line ABOVE the signature, not
  trailing (golines flags the trailing form).
- **gci grouping:** std group, then ONE default group with module + external
  imports alphabetical (`larsartmann` < `onsi`) — bit us in 2 files.
- **exhaustruct:** full literals even when all-zero (`PassResult{}` is
  flagged; all 6 fields explicit).
- Consider a pre-commit local run of `golangci-lint` on changed dirs —
  BuildFlow's lint step runs at commit time but the daemon can commit first.

## f) NEXT UP TO 50 (realistic queue, in order)

1. Fix 3 T10 lint findings; verify lint=0; fixup commit
2. T11: `reviewd.Replay(store, writer)` core (fold streams → re-render all markdown)
3. T11: `events` CLI body (+ remove stub nolint)
4. T11: `replay` CLI body
5. T11: replay test (pass → wipe reviews → replay → byte-diff INDEX/views)
6. T11: events output format (one line per event: seq, stream, type, sha, score)
7. T11: events filters (`-project`, `-view`, `-type`, `-last N`)
8. T12: fake `/v1/chat/completions` httptest server (content parts w/ image_url)
9. T12: E2E provider→reviewer over real HTTP incl. score parse
10. T12: E2E compare path over real HTTP
11. T13: doctor — config load check
12. T13: doctor — data/reviews dir writability
13. T13: doctor — glob match counts per project
14. T13: doctor — `GET {BaseURL}/models` contains configured Model
15. T13: doctor exit code reflects failures; tests
16. T14: flake.nix `visionreviewd` package (buildGoModule, ldflags version)
17. T14: fix stale vendorHash (fake-hash → build → real hash)
18. T14: `nix build .#visionreviewd` green
19. T15: NixOS module options (enable, config path, interval overrides)
20. T15: systemd unit with StateDirectory wiring
21. T15: optional llama-server unit (`-hf` model, port 8390) disabled by default
22. T15: module eval check (`nixos-option`/eval test)
23. T16: SystemNix lazy wrapper module file
24. T16: activation docs (input, lock, host enable, port registration)
25. T17: README daemon section (quickstart: discover → config → run)
26. T17: CHANGELOG entry
27. T17: AGENTS.md architecture notes for internal/reviewd + cmd/visionreviewd
28. T17: TODO_LIST/FEATURES rows
29. T17: example config JSON in docs
30. T18: `go test -race ./...` green
31. T18: vet + golangci + gofmt across repo green
32. Decide + execute push policy (default: session end)
33. Status report refresh after T11–T13
34. Consider `Interval()` accessor test / daemon options hardening
35. Consider per-view `ReviewedAt` in INDEX Updated column (currently CapturedAt)
36. Consider replay writing INDEX from folded state (may differ from pass-time INDEX)
37. Deduplicate PNG fixtures (scan_test/compare_test/BDD shotPNG copies)
38. Consider a `PassResult.String()` for CLI output
39. Guard: `Pipeline.Pass` on cancelled ctx — verify skip semantics
40. Consider blob GC (orphans after replay pruning) — document as future
41. Check `go.sum`/`go mod tidy` after all tasks
42. Update `docs/DUPLICATION_POLICY.md` if clone count changed

## g) QUESTIONS (cannot figure out ourselves)

1. **Push policy:** push each task commit to origin/master as it lands, or
   only at session end? (Only `c241fdc` is pushed; master is 10 commits
   ahead locally.)
2. **Reviews home for the example config (T17):** keep default
   `~/.local/share/vision-review-agent/reviews`, or ship the example
   pointing at a git-tracked repo (e.g. `~/projects/ui-reviews`) so Crush
   reads versioned reviews?
3. **llama-server unit in T15:** default-disabled as planned, or enabled by
   default on your NixOS host (model auto-pull is ~9–10 GB on first start)?
