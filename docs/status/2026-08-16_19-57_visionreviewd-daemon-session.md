# Status Report: visionreviewd Daemon — Session Snapshot

> **ANNOTATED 2026-08-16 (docs-health):** every T2–T18 task below shipped
> (see inline markers). Still open: real-model bring-up (#36) and DiscordSync
> baselines (#37) — tracked in `TODO_LIST.md`.

**Date:** 2026-08-16 19:57
**Scope:** Work done in this session (vision-review-agent → visionreviewd daemon transformation)
**Mission:** Turn the vision SDK into the event-sourced UI review daemon it was always meant to be: watch all projects' UI screenshots, review them with a local llama-server Qwen3-VL model, write Crush-consumable markdown reviews, record everything as events (go-cqrs-lite), auto-compare before→after.

---

## a) FULLY DONE

| Item                                                                 | Commit  | Evidence                                                                                                                                                                                                   |
| -------------------------------------------------------------------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Research: DiscordSync screenshot tooling                             | —       | `nix run .#gallery-shots` → `{Page}--{theme}--{viewport}.png`; committed goldens in `internal/web/testdata/visual/`                                                                                        |
| Research: SystemNix wiring contract                                  | —       | Daemons enter as `github:LarsArtmann/<repo>` flake inputs + lazy wrapper in `modules/nixos/services/`; local `path:` inputs banned                                                                         |
| Research: go-cqrs-lite v4 API surface                                | —       | `decider.Repository`, `event.Single`, `ParseStreamID/Type`, bbolt backend `Open(path, logger)`; all needed modules tagged remotely (event/v4.7.0, decider/v4.3.0, storage/v4.7.1, bbolt/v4.0.0, id/v4.5.0) |
| Research: llama-server model path via SDK                            | —       | `openaicompat.New(WithBaseURL(...))` → `LanguageModel` → `vision.NewAgent` — zero SDK changes needed                                                                                                       |
| Environment audit                                                    | —       | Port 8080 occupied by non-llama-server process; `llama-server` v10273 + `d2` installed; HF cache does NOT contain the 9–10GB model; no live OpenAI-compatible server                                       |
| Pareto execution plan (18 medium / 46 fine tasks, 4 tiers, D2 graph) | f0e9a61 | `docs/planning/archived/2026-08-16_19-47-visionreviewd-daemon-plan.html` + `visionreviewd-execution-graph.{d2,svg}`                                                                                        |
| **T1: go-cqrs-lite v4 dependency + bbolt spike**                     | 056ddac | `internal/reviewd/spike_test.go` — append 2 `view.captured` events via decider repository, fold state, reload raw events; PASS                                                                             |

**Test state:** `internal/reviewd` spike test green; full suite untouched this session (was green at session start).

## b) PARTIALLY DONE

| Item                                                                  | State                                                                                                                                              | What remains                                                                                                                                                                                                 |
| --------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| ~~**T2: Domain events, viewKey parsing, hashing**~~ done at `df51b84` | Research done (StreamType is `type StreamType string`, `ParseStreamType` non-error for constants; StreamID needs `ParseStreamID` + error handling) | ~~Write `events.go` / `views.go` / `hash.go` + table tests; replace spike payloads with real domain types~~ shipped (files landed as `events.go` / `viewkey.go` / `hash.go` / `blobstore.go`; spike deleted) |

## c) NOT STARTED

~~T3 Config + discover · T4 Reviewer (llama-server prompts, score) · T5 Markdown writer (views/comparisons/INDEX) · T6 bbolt store wiring + View decider · T7 Review pipeline + BDD · T8 A/B compare · T9 CLI (7 subcommands) · T10 Daemon loop + SIGTERM · T11 events/replay commands · T12 E2E fake OpenAI server · T13 doctor · T14 flake package · T15 NixOS module (this repo) · T16 SystemNix lazy wrapper · T17 Docs · T18 Final verification (race/vet/lint/fmt).~~ ALL DONE: `b9d873a` (T3), `681915e`+`47a4137` (T4), `b24561a` (T5), `d48c947` (T6), `f9e77dd` (T7), `4e8316d` (T8 auto/manual), `660e9d3` (T9), `de7c4c6` (T10), `5cddff4` (T11), `99dcdfe` (T12), `cee0ba7` (T13), `9fd3117` (T14), `b2d9c0c` (T15), `4c03af9`+`df7cdf2` (T16), `5af6d22` (T17), all gates green (T18).

## d) TOTALLY FUCKED UP!

Nothing permanently broken. Friction hits (all resolved):

1. **Commit f0e9a61 first attempt blocked by BuildFlow** — inlined D2 SVG contains CDATA `<style>` which prettier cannot parse. Fixed by switching to the house pattern `<object data="visionreviewd-execution-graph.svg">` (external ref, file committed alongside) + running `prettier --write`. Hook then passed.
2. **`go mod tidy` silently dropped the cqrs deps** before any file imported them (tidy between `go get` and first import). Re-added; lesson recorded: do not tidy before imports exist.
3. Pre-existing (not session-caused): LSP/golangci-lint cannot init build cache at `/mnt/buildcache` — diagnostics are garbage; CLI `go build`/`go test` unaffected.

## e) WHAT WE SHOULD IMPROVE!

- **Fix the LSP build cache env** (`GOCACHE` → unavailable mount) so in-editor diagnostics stop lying.
- **Prompt quality is untested against the real model** — the pinned model is caption-tuned; review prompts must lean descriptive→critical. Needs a real-model tuning session once llama-server is up (blocked on 9–10GB download).
- **Port hygiene**: llama-server default 8080 is taken on this host; the NixOS module must take an explicit port (register in SystemNix `lib/ports.nix`).
- **Reviews location permanence**: decide state-dir vs dedicated git repo (open question g1) before T5 lands, so the layout doesn't churn.
- Keep the BuildFlow pre-commit in mind for every commit (it auto-formats; ~30–60s per commit — batch changes where sensible).

## f) Top things to get done next (impact-sorted)

1. ~~T2: `events.go` — real domain payloads (`Captured/Reviewed/Compared`) + event type constants~~ done at `df51b84`
2. ~~T2: `views.go` — viewKey parser (`{Page}--{theme}--{viewport}` + fallback) + tests~~ done at `df51b84` (landed as `viewkey.go`)
3. ~~T2: `hash.go` — streaming sha256 file hash + tests~~ done at `df51b84`
4. ~~T2: `ViewStreamID(project, viewKey)` helper + tests~~ done at `df51b84`
5. ~~T2: delete spike test (superseded by real store tests in T6)~~ done at `df51b84`
6. ~~T3: Config struct + defaults + validation (offending values in errors)~~ done at `b9d873a`
7. ~~T3: `~` path expansion + tests~~ done at `b9d873a`
8. ~~T3: `discover` walker (testdata/visual, gallery-shots patterns) → suggested JSON config~~ done at `b9d873a`
9. ~~T4: provider ctor (`openaicompat.WithBaseURL`) + model id wiring~~ done at `681915e`
10. ~~T4: review prompt template (scored markdown output)~~ done at `47a4137`
11. ~~T4: A/B compare prompt template (improved/worsened sections)~~ done at `47a4137`
12. ~~T4: `Reviewer.Review` via `vision.Analyze` + mock-model test~~ done at `47a4137`
13. ~~T4: score extraction (`Score: N/10` regex) + tests~~ done at `47a4137`
14. ~~T5: view markdown render + test~~ done at `b24561a`
15. ~~T5: comparison markdown render + test~~ done at `b24561a`
16. ~~T5: `INDEX.md` aggregation + trend arrows + test~~ done at `b24561a`
17. ~~T6: store open/close wiring from config path~~ done at `d48c947`
18. ~~T6: View decider fold (`ViewState`) + tests~~ done at `d48c947`
19. ~~T6: journal scan → changed-views projection + tests~~ done at `d48c947`
20. ~~T7: scanner (globs → viewKey/path/hash)~~ done at `f9e77dd`
21. ~~T7: pipeline pass orchestration~~ done at `f9e77dd`
22. ~~T7: BDD full pass (mock model) → events + files asserted~~ done at `f9e77dd`
23. ~~T8: auto-compare on changed capture vs previous~~ done at `f9e77dd` (wired into the T7 pipeline)
24. ~~T8: manual compare core + CLI~~ done at `4e8316d`, `660e9d3`
25. ~~T9: subcommand skeleton (testable, no os.Exit)~~ done at `660e9d3`
26. ~~T10: daemon loop + context/SIGTERM + single-flight~~ done at `de7c4c6`
27. ~~T11: `events` + `replay` (rebuild reviews dir from log)~~ done at `5cddff4`
28. ~~T12: fake OpenAI httptest server + E2E reviewer test~~ done at `99dcdfe`
29. ~~T13: `doctor` (config, dirs, `/v1/models`, model id match)~~ done at `cee0ba7`
30. ~~T14: flake `visionreviewd` package + `nix build` verify~~ done at `9fd3117`
31. ~~T15: NixOS module in this repo (+ optional llama-server unit, explicit port)~~ done at `b2d9c0c`
32. ~~T16: SystemNix lazy wrapper + activation doc (input, lock, host enable)~~ done at `4c03af9` (SystemNix `8fc2b80c`) — host enable + lock bump still open (TODO_LIST)
33. ~~T17: README/CHANGELOG/AGENTS/TODO_LIST/FEATURES updates~~ done at `5af6d22`
34. ~~T18: full `go test -race ./...`, vet, golangci-lint, gofmt~~ done — all green at session close
35. ~~Post-session: push repo → SystemNix lock bump → enable on host~~ repo pushed (remote master = `5da8022`); SystemNix lock bump + host enable still open (TODO_LIST)
36. Post-session: download model, run llama-server, real-model prompt tuning session ← still open (TODO_LIST)
37. Post-session: DiscordSync baselines into default config; consider CI hook (gallery-shots → daemon ingest) ← still open (TODO_LIST)

## g) Questions I can NOT figure out myself

1. ~~**Reviews home:** default `~/.local/share/vision-review-agent/reviews/` (state dir, planned) or a dedicated **git-tracked repo** (e.g. `~/projects/ui-reviews`) so review history is versioned and you can point Crush at a repo? This shapes T5/T9 layout — decide before the writer lands.~~ resolved by plan v2 decision 1 — default state dir; `reviewsDir` is configurable so a git repo works without code changes
2. ~~**llama-server bring-up in SystemNix:** should the module auto-download the ~9–10GB GGUF at activation (`-hf`, needs network/disk on the NixOS host), or do you run llama-server manually/elsewhere and the module only manages `visionreviewd` pointing at an existing base URL?~~ resolved — module ships an optional, default-disabled llama-server unit with `-hf` auto-pull (`b2d9c0c`)
3. ~~**Model default:** keep `GitMylo/nsfwcaption-qwen3-vl-8b-v3-gguf:Q8_0` hardcoded as the config default (my plan) even though it is caption-tuned (prompts must compensate), or ship no default and always configure explicitly?~~ resolved — kept as config default per user specification; prompts engineered descriptive→critical

---

_Point-in-time snapshot. Living task tracking stays in `TODO_LIST.md` per project docs policy._
