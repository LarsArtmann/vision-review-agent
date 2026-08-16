# Status Report: visionreviewd Daemon — Session Snapshot

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
| Pareto execution plan (18 medium / 46 fine tasks, 4 tiers, D2 graph) | f0e9a61 | `docs/planning/2026-08-16_19-47-visionreviewd-daemon-plan.html` + `visionreviewd-execution-graph.{d2,svg}`                                                                                                 |
| **T1: go-cqrs-lite v4 dependency + bbolt spike**                     | 056ddac | `internal/reviewd/spike_test.go` — append 2 `view.captured` events via decider repository, fold state, reload raw events; PASS                                                                             |

**Test state:** `internal/reviewd` spike test green; full suite untouched this session (was green at session start).

## b) PARTIALLY DONE

| Item                                            | State                                                                                                                                              | What remains                                                                                            |
| ----------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| **T2: Domain events, viewKey parsing, hashing** | Research done (StreamType is `type StreamType string`, `ParseStreamType` non-error for constants; StreamID needs `ParseStreamID` + error handling) | Write `events.go` / `views.go` / `hash.go` + table tests; replace spike payloads with real domain types |

## c) NOT STARTED

T3 Config + discover · T4 Reviewer (llama-server prompts, score) · T5 Markdown writer (views/comparisons/INDEX) · T6 bbolt store wiring + View decider · T7 Review pipeline + BDD · T8 A/B compare · T9 CLI (7 subcommands) · T10 Daemon loop + SIGTERM · T11 events/replay commands · T12 E2E fake OpenAI server · T13 doctor · T14 flake package · T15 NixOS module (this repo) · T16 SystemNix lazy wrapper · T17 Docs · T18 Final verification (race/vet/lint/fmt).

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

1. T2: `events.go` — real domain payloads (`Captured/Reviewed/Compared`) + event type constants
2. T2: `views.go` — viewKey parser (`{Page}--{theme}--{viewport}` + fallback) + tests
3. T2: `hash.go` — streaming sha256 file hash + tests
4. T2: `ViewStreamID(project, viewKey)` helper + tests
5. T2: delete spike test (superseded by real store tests in T6)
6. T3: Config struct + defaults + validation (offending values in errors)
7. T3: `~` path expansion + tests
8. T3: `discover` walker (testdata/visual, gallery-shots patterns) → suggested JSON config
9. T4: provider ctor (`openaicompat.WithBaseURL`) + model id wiring
10. T4: review prompt template (scored markdown output)
11. T4: A/B compare prompt template (improved/worsened sections)
12. T4: `Reviewer.Review` via `vision.Analyze` + mock-model test
13. T4: score extraction (`Score: N/10` regex) + tests
14. T5: view markdown render + test
15. T5: comparison markdown render + test
16. T5: `INDEX.md` aggregation + trend arrows + test
17. T6: store open/close wiring from config path
18. T6: View decider fold (`ViewState`) + tests
19. T6: journal scan → changed-views projection + tests
20. T7: scanner (globs → viewKey/path/hash)
21. T7: pipeline pass orchestration
22. T7: BDD full pass (mock model) → events + files asserted
23. T8: auto-compare on changed capture vs previous
24. T8: manual compare core + CLI
25. T9: subcommand skeleton (testable, no os.Exit)
26. T10: daemon loop + context/SIGTERM + single-flight
27. T11: `events` + `replay` (rebuild reviews dir from log)
28. T12: fake OpenAI httptest server + E2E reviewer test
29. T13: `doctor` (config, dirs, `/v1/models`, model id match)
30. T14: flake `visionreviewd` package + `nix build` verify
31. T15: NixOS module in this repo (+ optional llama-server unit, explicit port)
32. T16: SystemNix lazy wrapper + activation doc (input, lock, host enable)
33. T17: README/CHANGELOG/AGENTS/TODO_LIST/FEATURES updates
34. T18: full `go test -race ./...`, vet, golangci-lint, gofmt
35. Post-session: push repo → SystemNix lock bump → enable on host
36. Post-session: download model, run llama-server, real-model prompt tuning session
37. Post-session: DiscordSync baselines into default config; consider CI hook (gallery-shots → daemon ingest)

## g) Questions I can NOT figure out myself

1. **Reviews home:** default `~/.local/share/vision-review-agent/reviews/` (state dir, planned) or a dedicated **git-tracked repo** (e.g. `~/projects/ui-reviews`) so review history is versioned and you can point Crush at a repo? This shapes T5/T9 layout — decide before the writer lands.
2. **llama-server bring-up in SystemNix:** should the module auto-download the ~9–10GB GGUF at activation (`-hf`, needs network/disk on the NixOS host), or do you run llama-server manually/elsewhere and the module only manages `visionreviewd` pointing at an existing base URL?
3. **Model default:** keep `GitMylo/nsfwcaption-qwen3-vl-8b-v3-gguf:Q8_0` hardcoded as the config default (my plan) even though it is caption-tuned (prompts must compensate), or ship no default and always configure explicitly?

---

_Point-in-time snapshot. Living task tracking stays in `TODO_LIST.md` per project docs policy._
