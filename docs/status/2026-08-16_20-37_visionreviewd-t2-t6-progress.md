# visionreviewd build-out — T2–T6 progress snapshot

**Date:** 2026-08-16 20:37 CEST
**Mission:** Turn vision-review-agent into the event-sourced UI review daemon
(visionreviewd) per
[`docs/planning/2026-08-16_20-00_visionreviewd-full-execution-plan-v2.md`](../planning/2026-08-16_20-00_visionreviewd-full-execution-plan-v2.md)
(T2–T18).
**Authoritative plan:** plan v2 (committed c241fdc, pushed).

## What was done this session (all verified green at their commit)

| Task | Content                                                                                                                                                                                                                                                                                               | Commit    |
| ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- |
| T2   | Domain: event payloads (`view.captured/reviewed/compared`), `ParseViewKey` with deterministic fallbacks, streaming `SHA256File`, content-addressed `BlobStore`; spike test deleted                                                                                                                    | `df51b84` |
| T3   | `Config` (JSON durations as strings, `~` expansion, validation with offending values), `LoadConfig`, `DiscoverProjects` walker (testdata/visual, gallery-shots, screenshots, ui-screenshots) + paste-ready JSON; default llama port **8390** (verified free in SystemNix `lib/ports.nix`; 8080 taken) | `b9d873a` |
| T4   | `Reviewer` via fantasy openaicompat provider + vision SDK; caption-tuned prompts (describe→judge, mandatory `Score: N/10` final line); `ExtractScore` (bold/case/space tolerant, last-match-wins); `Compare` A→B (rates AFTER); mock `fantasy.LanguageModel` tests                                    | `47a4137` |
| T5   | Markdown renderer (view review, comparison, INDEX with score table + trend arrows ▲▼▬·), `Writer` with atomic temp+rename writes, project-name sanitization (path-traversal safe), UTC timestamps; plus repo-lint hardening of T2/T3                                                                  | `b24561a` |

Every commit passed `go build ./...`, `go test ./internal/reviewd/`,
`golangci-lint run ./internal/reviewd/...` (0 issues), and the BuildFlow
pre-commit hook.

## Current state: T6 IN PROGRESS, RED, uncommitted

`internal/reviewd/store.go` + `store_test.go` (new, untracked) implement the
bbolt event store: `OpenStore/Close`, `ViewState` fold for the 3 event types,
`RecordCapture/RecordReview/RecordComparison`, `LoadView`, `ViewEvents`,
`AllEvents` (journal `ReadAll`).

**1 test fails right now:**

```
--- FAIL: TestStoreScoreTrend
    store_test.go:165: after first review: last=6 prev=-1, want 6/0
```

Cause: while satisfying `exhaustruct`, `initialViewState.LastScore/PrevScore`
were set to `ScoreUnknown` (-1) instead of 0. The new behavior is arguably
CORRECT (no prior review = unknown trend), so the fix is updating the test
expectation to `PrevScore == ScoreUnknown` after the first review — not
reverting the initializer.

**3 lint findings remain in T6 files:**
`gochecknoglobals` on `initialViewState`, `errorlint` on the
`(close error: %s)` verb, one `golines` long line.

## Mistakes made & lessons

- `json.Unmarshal` returns 1 value — wrote a 2-value call in events_test first.
- Invented `decodeOver()` helper that never existed (config test).
- 2-part view names mis-parsed (`Settings--dark` → `Settings--dark--Settings--dark`); fixed via explicit `case 2`.
- `id.ParseStreamID` accepts ANY non-empty string — empty project would
  silently produce `:view` streams; added explicit `ErrEmptyProject` guard.
- discover.go v1 lost project dir case (`DiscordSync`→`discordsync` glob
  paths broke); rewrote with accumulator tracking real paths.
- fantasy APIs verified by reading module source: `Prompt` is `[]Message`
  (no `.Parts()`), images travel as `FilePart`, `LanguageModel` interface
  needs `Provider() string` + `Model() string`, `NewAgent` returns `(*Agent, error)`.
- `len(const)` compile error in prompts.go (scoreRules signature churn ×3).
- sed-injected duplicate `t.Parallel()` → test panic.
- `time.Local` banned by gosmopolitan → all display timestamps are UTC now.
- **Repeat-offender lesson:** BuildFlow gofmt-touches files between my read
  and edit → several "file modified since read" aborts; re-view before edit.
- go-cqrs-lite `system/` package deliberately NOT used (full runtime
  framework; daemon needs append+fold+journal only). Plan v2's lean core is
  the right size.

## Runtime verification results

- `go test ./...` — all pre-existing packages green (cached) after each task.
- reviewd package: 0 golangci-lint issues at T2–T5 commits.
- `.golangci.yaml`: added `github.com/larsartmann/go-cqrs-lite` to depguard
  allowlist (required; config change committed with T5).
- Real llama-server E2E deferred (model ~9–10 GB not in HF cache) — T12
  covers the HTTP path with a fake OpenAI-compatible server.

## Files added this session (beyond plan's names)

`internal/reviewd/`: `events.go`, `viewkey.go`, `hash.go`, `blobstore.go`,
`config.go`, `discover.go`, `provider.go`, `prompts.go`, `reviewer.go`,
`score.go`, `markdown.go`, `writer.go`, `store.go` (+ `_test.go` each).
`.golangci.yaml` (depguard allow).

## Branch & push state

- Branch `master`, HEAD `b24561a`. All task commits are LOCAL only.
- Pushed earlier this session: plan v2 (`c241fdc`). Remote HEAD = `c241fdc`.
- Auto-git daemon + BuildFlow active; hook occasionally reformats files
  post-commit (next task picks the diffs up — nothing lost).

## What is NOT done yet (T6 remainder + T7–T18)

- T6: fix test expectation, clear 3 lint findings, commit.
- T7 pipeline (scan→ingest→compare→review→write + INDEX refresh + BDD),
- T8 auto/manual compare flows, T9 CLI (7 subcommands), T10 daemon loop +
  SIGTERM, T11 events/replay, T12 fake-server E2E, T13 doctor,
  T14 flake package (+stale `vendorHash` at flake.nix:52), T15 NixOS module
  (+optional llama-server unit on 8390), T16 SystemNix lazy wrapper +
  activation docs, T17 docs, T18 final `-race`/vet/lint/gofmt gate.

## Open questions (max 3)

1. **Push policy:** push each task commit to origin/master as it lands, or
   only on explicit request / session end?
2. **Reviews home:** keep default
   `~/.local/share/vision-review-agent/reviews`, or point the shipped
   example config at a dedicated git-tracked repo (e.g.
   `~/projects/ui-reviews`) from day one?
3. **INDEX trend baseline:** OK that the first-ever review of a view shows
   trend "·" (unknown previous score) rather than pretending 0?

**Next action (awaiting go-ahead):** fix `TestStoreScoreTrend` expectation +
the 3 lint findings, commit T6, then proceed to T7.
