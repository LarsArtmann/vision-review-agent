<!-- visionreviewd repo · point-in-time status snapshot · do not rewrite, annotate instead -->

# Status Report · 2026-08-29 18:36 CEST · Prompt-Contract Hardening Session

Session scope: comparison research (Hyper-Extract, paperless-ngx AI) and the
`internal/reviewd/prompts.go` improvement task. Nothing outside this session is
claimed as verified state — per the standing rule, point-in-time claims about
older work would need re-verification before being trusted.

---

## 0 · What this session actually did

| # | Work | Outcome |
|---|------|---------|
| 1 | User-supplied URL `yukimura923/audit-o-tron` | Verified non-existent (404 repo AND 404 owner, GitHub search 0 hits). Refused to fabricate a comparison. |
| 2 | `yifanfeng97/Hyper-Extract` research + comparison vs vision-review-agent & paperless-ngx | Real differentiators identified (hypergraphs, YAML template layer, incremental graph evolution); comparison table delivered; steal-ideas noted (MCP server, Obsidian export, template-driven prompts). |
| 3 | `prompts.go` explained on request | Mapped all 6 pieces and their consumers. |
| 4 | paperless-ngx AI updates research | v3.0.0 (2026-07-22) shipped built-in Paperless AI (suggestions, RAG chat, sqlite-vec, Ollama/OpenAI); v3.1.0 added workflow-action suggestions + selective remote OCR. No 2025 release had AI. |
| 5 | **Main task: improve `prompts.go`** (READ→UNDERSTAND→RESEARCH→REFLECT→execute loop) | See sections 1–3. |

---

## a) FULLY DONE

1. **Split-brain bug fixed: unwired `CompareSystemPrompt`** — the const existed,
   its doc claimed it was "the system prompt for A/B comparisons", but
   `NewReviewer` built ONE agent under the review persona, so every BEFORE/AFTER
   compare silently ran under the single-view system prompt.
   `Reviewer` now builds two agents (`reviewAgent` / `compareAgent`), one per
   persona (`internal/reviewd/reviewer.go:19,56`), with a `newAgent` helper
   that keeps the wrapcheck-clean `"create review/compare agent: %w"` errors.
2. **Prompt contract pinned by tests** — new `internal/reviewd/prompts_test.go`
   (previously ZERO direct tests for the prompt builders):
   - view-context prefix, the exact `## Summary → ## Strengths → ## Issues →
     ## Recommendations` heading ORDER (the prompt demands "EXACTLY this
     markdown structure" — now the contract is enforced both ways),
   - compare prompt: `Image 1 BEFORE / Image 2 AFTER` ordering line, its own
     heading order, "rating the AFTER (second) image" wording, and a guard
     that single-review sections (`## Summary`) cannot leak into compare,
   - the literal `Score: N/10` final-line contract and `integer 0-10`.
3. **Persona-wiring regression test** —
   `TestReviewerUsesPersonaSystemPrompts` proves via the captured mock prompt
   that review runs under "judge them like a senior product designer" and
   compare under "comparing two versions of the same view", each WITHOUT the
   other's persona. Also empirically confirmed the system prompt actually
   reaches the model in `fantasy.Call.Prompt`.
4. **Root-cause regex fix in `score.go`** — `StripScoreLines` left a stray
   `**` residue in rendered review files for bolded score lines
   (`**Score: 8/10**`) because the regex tolerated only LEADING bold markers.
   Trailing `\**` added (`internal/reviewd/score.go:13`); extraction semantics
   unchanged (last match wins, capture group untouched).
5. **`StripScoreLines` test coverage** — 5-case table added to
   `score_test.go` (contract line, bolded, multiple lines, non-contract
   wording kept, empty). It was consumer-loaded (`markdown.go:47`) and untested.
6. **`scoreRules` deduplication** — the two near-identical branches collapsed
   into one conditional suffix; output verified byte-identical
   (`internal/reviewd/prompts.go:78`).
7. **Dead code removal** — unused `writeFile` helper in `score_test.go`
   (flagged by gopls `unusedfunc`) deleted with its orphaned imports.
8. **AGENTS.md updated** (standing memory rule, done at discovery time):
   - new Key Design Decision bullet under visionreviewd (two-agent persona
     wiring + pinned contract + regex residue fix),
   - architecture line for `reviewer.go` corrected to "two agents, one per
     persona",
   - new gotcha bullet: build caches pointing at `/mnt/buildcache`.
9. **Verification run** — `go build ./...`, `go vet ./...`, `gofmt -l .`
   (clean), full `go test ./...` (exit 0, jsonv2 regime active locally),
   `go test -race ./internal/reviewd/` (green).

## b) PARTIALLY DONE

1. **Lint-clean package** — `golangci-lint run ./internal/reviewd/...` exits 1
   on ONE finding: `config.go:34` unused `//nolint:recvcheck` directive. That
   file is untouched by this session and the finding reproduces on unmodified
   HEAD; the local golangci-lint is NEWER than the flake-pinned one (it also
   warns `exhaustruct` is deprecated → `exhaustruct_v5`). Deliberately not
   "fixed" blind — needs the pinned toolchain verdict (see question 2).
2. **Verification matrix** — ran: items 1 (build/vet/gofmt), 3 (jsonv2
   build+vet+tests, since local `GOEXPERIMENT=jsonv2`), full non-race suite.
   NOT run: full `-race ./...` (only reviewd), item 4 (`GOEXPERIMENT=none` SDK
   subset — defensible: SDK untouched, daemon is jsonv2-only by design),
   items 5–8 (`go mod verify`, `tidy -diff`, nix test/lint/build, flake
   check). Judged disproportionate for a daemon-internal change, but
   honestly: not verified.
3. **CHANGELOG** — no `0.7.0-dev` Unreleased entry written for the persona fix
   yet; AGENTS.md records it, the changelog does not.
4. **Commit state** — nothing committed by me (critical rule: no commit unless
   explicitly asked). Working tree: 6 code/doc files modified, 1 new test
   file, this report. The auto-commit daemon may pick it up.

## c) NOT STARTED (discussed this session, deliberately parked)

1. **Structured-output reviews** — replace the free-markdown + regex score
   contract with `AnalyzeStructured[ReviewResult]` (score as int, sections as
   typed fields). The SDK already has the machinery; the daemon doesn't use it.
2. **YAML/template-driven prompt presets** (the Hyper-Extract lesson) —
   `prompts.go`'s two hardcoded shapes generalized into per-view-type
   templates.
3. **MCP server exposing visionreviewd reviews** (`he-mcp` pattern).
4. **Obsidian export** of the views/comparisons markdown projections.
5. paperless-ngx ↔ Hyper-Extract layering idea — external-repo musing only;
   no action belongs to this repo.

## d) TOTALLY FUCKED UP

Nothing catastrophic. Honest near-misses:

1. **My first `StripScoreLines` test expectation was written optimistically**
   (`want: "text"` before checking actual behavior). The failing run is what
   exposed the trailing-bold regex gap — a good outcome reached by accident
   rather than by a deliberate failing-test-first step. Process luck, not
   process discipline.
2. **Two wasted round trips on the environment**: `GOCACHE`/`GOMODCACHE`
   pointing at `/mnt/buildcache` ("no such device") failed the first two
   build attempts before I overrode them. The gotcha is now in AGENTS.md, but
   it cost attempts that a known-gotcha check would have saved.
3. **I left the lint exit-1 standing** instead of settling it with
   `nix run .#lint` (pinned toolchain). Defensible (don't touch untouched
   files on tool-version suspicion), but it means the session ends with a
   known red exit code and an open question instead of a resolution.
4. **Never checked `git log` mid-session** — the standing docs say an
   auto-commit daemon runs continuously; I never verified whether it
   interfered or produced commits during the session. Unknown, unchecked.

## e) WHAT WE SHOULD IMPROVE

1. **Verify, don't assume, with the pinned toolchain** — the config.go nolint
   question and the `exhaustruct_v5` migration both need one
   `nix run .#lint` run to settle.
2. **Kill the score-contract regex entirely** (see c-1): every contract point
   currently enforced by prompt wording + a tolerant regex would become a
   typed struct field; `ExtractScore`/`StripScoreLines` and their failure
   modes (`ScoreUnknown`, `?` in INDEX) disappear.
3. **Prompt drift detection** — prompts are now pinned by tests, but nothing
   alerts when a real model answer deviates from the pinned sections (e.g.
   missing `## Issues`); a cheap log/INDEX signal would surface model regressions.
4. **Doc drift hygiene for AGENTS.md architecture lines** — my own
   `reviewer.go` line went stale within one refactor; the architecture block
   should be checked whenever a listed file's shape changes (a docs-health
   VERIFY pass habit).
5. **Commit/report loop** — this report's section (f) is HARVEST input for
   `TODO_LIST.md`/`ROADMAP.md`; leaving it entombed here would repeat a known
   anti-pattern.
6. **Redundant compare framing** — compare persona now stated in BOTH system
   prompt and user prompt; harmless for caption-tuned models, worth one
   look during the next prompt-quality iteration (with a real eval, not vibes).

## f) Up to 50 things to get done next

Brainstorm list, sorted roughly by impact × effort; most of items 10+ are
ROADMAP fuel, not commitments (HARVEST must apply routing rigor).

| # | Item | Bucket |
|---|------|--------|
| 1 | Run `nix run .#lint` to settle config.go `nolint:recvcheck` + decide linter pin bump (`exhaustruct_v5`) | verify |
| 2 | Complete verification matrix after session changes (`-race ./...` full, `go mod verify`, `tidy -diff`, `nix run .#test`, `nix build .#visionreviewd`, `nix flake check`) | verify |
| 3 | CHANGELOG entry under 0.7.0-dev: compare persona fix + score-strip residue fix | docs |
| 4 | HARVEST this report's (f) into `TODO_LIST.md` / `ROADMAP.md` | docs |
| 5 | Commit session work (or confirm auto-daemon did, with a sane message) | housekeeping |
| 6 | Spike: `AnalyzeStructured[ReviewResult]` for reviews (typed score/sections), keep markdown rendering as projection | refactor |
| 7 | Model-answer deviation warning (log or INDEX hint when pinned sections are missing) | quality |
| 8 | Wire `Config.Retry` in the daemon for retryable model failures (SDK supports it; daemon sets nothing) | robustness |
| 9 | Record token usage per review/compare in events (fantasy `Usage` is available; events currently don't carry it) | feature |
| 10 | Cost tracking in the daemon via `CostTracker` + surface cost in INDEX | feature |
| 11 | Prompt eval harness: small golden screenshot set scored before/after prompt edits | quality |
| 12 | Golden-file test pinning full prompt bytes (catch accidental wording drift beyond structure) | tests |
| 13 | Fuzz `ExtractScore`/`StripScoreLines` (repo already has fuzz culture in a2ui) | tests |
| 14 | Idempotence property test: `StripScoreLines` ∘ `ExtractScore` composition | tests |
| 15 | Per-view-type prompt presets via config (YAML template layer) | ROADMAP |
| 16 | MCP server exposing reviews/comparisons to agents (`he-mcp` pattern) | ROADMAP |
| 17 | Obsidian vault export of the markdown projections | ROADMAP |
| 18 | INDEX.md: last-error column / error kind when a review failed (today: bare `?`) | UX |
| 19 | Doctor subcommand: model reachability + one dry-run review validating the contract | feature |
| 20 | Include previous score + prior issues in compare prompt (context-aware reviews) | quality |
| 21 | Theme/viewport-specific review guidance (dark-contrast, mobile-overflow) | quality |
| 22 | Output-language option for reviews (paperless parity: respect-language) | ROADMAP |
| 23 | Parallel per-view reviews with bounded semaphore — AFTER verifying `vision.Agent` concurrency safety | perf |
| 24 | Document `ExtractScore` tolerance rules in docs (what strings parse, why) | docs |
| 25 | Reviewer-as-interface (mirror the `PassRunner` pattern) for consumer substitution | refactor |
| 26 | BDD specs for the prompts contract if it gains behavior (currently table tests suffice) | tests |
| 27 | Dogfood `pkg/vision/a2ui`: render a review as an A2UI surface | ROADMAP |
| 28 | Score-history sparkline in view files (events already hold the data) | feature |
| 29 | Retry/backoff parity between auto-compare failure path and review path | robustness |
| 30 | `visionreviewd once` test coverage for missing-predecessor-blob compare path | tests |
| 31 | Check `DOMAIN_LANGUAGE.md` covers "persona", "score contract", "projection" terms | docs |
| 32 | Escape/`sanitizeCell`-review: model markdown is injected raw into view files — document the trust model | docs |
| 33 | Better hint in logs when score is missing (point at the contract line) | UX |
| 34 | CI job: lint-version parity check between local dev and flake pin | ci |
| 35 | Examples: end-to-end visionreviewd example project in `examples/` | docs |
| 36 | Silence/justify the stale `wsl_v5` LSP diagnostic noise (tooling hygiene) | housekeeping |
| 37 | Decide commit policy for agent sessions: manual vs trust-the-auto-daemon | policy |
| 38 | `newAgent` label param: consider typed persona enum if a third persona ever appears | nit |
| 39 | Config validation: friendly error when `PAPERLESS`-style AI env vars are half-set (openaicompat parity check) | UX |
| 40 | INDEX refresh under dead context: covered — add regression note in AGENTS.md test section? verify first | verify |
| 41 | Audit whether `Reviewer` should carry per-call timeout override (config.Timeout is global) | design |
| 42 | Add `visionreviewd replay` fuzz/property test for journal↔markdown determinism beyond existing BDD | tests |
| 43 | Compare file header: add score bullet + strip there too (today only view files strip; compare keeps the line by design — document why in DOMAIN_LANGUAGE if kept) | docs |
| 44 | Prompt i18n of the fixed section headings (English-only contract today) | ROADMAP |
| 45 |events payload: include model persona (review vs compare) explicitly | feature |
| 46 | Housekeeping: `go mod verify` + `tidy -diff` as a pre-commit hook candidate | ci |
| 47 | Investigate whether `GOEXPERIMENT=none` SDK job should also cover `internal/reviewd` fakeserver tests (currently daemon-excluded by design — confirm still correct) | verify |
| 48 | Screenshot dedup: skip review when SHA unchanged AND score exists (config-gated) | feature |
| 49 | README: one-line daemon quickstart pointing at the NixOS module | docs |
| 50 | Schedule a docs-health VERIFY pass over AGENTS.md claims older than this session | docs |

## g) Questions I can NOT figure out myself

1. **Structured output direction**: should the review/compare contract move
   from free-markdown + `Score: N/10` regex to `AnalyzeStructured[ReviewResult]`
   (typed score, sections as fields)? This is a product decision: it changes
   provider requirements (structured-output support needed from
   llama-server-class endpoints), streaming behavior, and the markdown
   projection pipeline. I cannot decide the flexibility-vs-typing tradeoff
   for the daemon's actual usage.
2. **Linter pin policy**: is the flake-pinned golangci-lint version
   intentionally frozen? Specifically: may I bump it (absorbing
   `exhaustruct_v5` and the now-dead `//nolint:recvcheck` at
   `config.go:34`), or must the pin stay and those findings be treated as
   local-tool noise?
3. **Roadmap intent**: were the Hyper-Extract steal-ideas (YAML prompt
   templates, MCP server for reviews, Obsidian export) discussion-only, or do
   you want any of them routed into `ROADMAP.md` as real intent?

---

*Report generated 2026-08-29 18:36 CEST. Format note: user explicitly
requested `.md` for this report, overriding the status-report skill's HTML
default. Nothing committed by the agent (no explicit commit instruction);
auto-commit daemon may have picked up changes.*
