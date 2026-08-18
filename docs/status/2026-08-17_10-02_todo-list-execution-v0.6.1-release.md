# TODO-List Execution + v0.6.1 Release — Session Report

> **ANNOTATED 2026-08-18 (docs-health):** b.2 (pkg.go.dev) resolved — v0.6.1
> is published and rendered there (verified 2026-08-18). c.4/f.2 (SystemNix
> lock commit) resolved — the lock pins `dcd50a0` and is committed. The g
> questions (tag deletion, v0.6.1 mismatch, Latest pointer) are consolidated
> into ROADMAP open question 5; the remaining open items live in TODO_LIST.

**Date:** 2026-08-17 (worked ~00:05–00:40, report written 10:02)
**Scope:** Full execution of TODO_LIST.md as of session start (20 items), ending in the v0.6.0/v0.6.1 releases.
**End state:** master `2cf3ced`, working tree clean, **CI green** (last 2 runs, first green since 2026-08-12/v0.5.0).

---

## a) FULLY DONE (verified)

### CI (critical) — both items closed

1. **`.golangci.yaml` wrapcheck schema fixed** — `ignoreSigs`/`ignore-type-assert-ok`
   (removed in the golangci-lint v2 schema) became `extra-ignore-sigs` (kebab-case;
   _adds_ to the default list instead of replacing it). `golangci-lint config verify`
   green locally (v2.12.2) **and** in CI. Four `//nolint:wrapcheck` directives that
   the fixed config made redundant were deleted (`pkg/vision/{retry,vision,structured}.go`).
2. **Daemon CI surface** — implemented as flake checks (so the existing
   `nix-flake-check` CI job covers them): `checks.visionreviewd` (package build),
   `checks.visionreviewd-version-smoke` (runs the binary — catches the
   silent-empty-build class), `checks.nixos-module-enabled`/`-disabled` (module
   evaluation both ways; enabled case forces the systemd ExecStart, deliberately
   NOT llama-cpp's). All four built and passed locally via `nix flake check`.

### Code quality (all 9 items closed)

3. **doctor stderr injection** — `checkModelEndpoint` now reads+closes the body
   eagerly and reports read/close failures through the check detail; no more
   direct `os.Stderr` writes bypassing injected writers.
4. **`doctorCheckExtra` killed** — capacity derives from the project count; the
   magic `3` is gone.
5. **INDEX "Updated" → review-aware** — new `ViewState.UpdatedAt()` (review time
   when the current capture was reviewed, else capture time) used by BOTH the
   pass-time INDEX refresh and Replay; 4-case table test proves replay
   determinism holds.
6. **`Pipeline.Pass` cancellation guard** — explicit "skipped, pass context done"
   errors (wrapped `ctx.Err()`) at project and view boundaries, INDEX refresh
   skipped under a dead context; two new tests (pre-cancelled + mid-pass cancel
   via a context-cancelling mock).
7. **exhaustruct excludes** — verified the TODO's premise was stale (0 findings
   at the time); my own refactor then introduced `reviewed.Config{}` literals, so
   excludes for `reviewd.Config/PassResult/ReplayResult` were added with a comment.
8. **codespell "unparseable"** — reworded to "malformed" everywhere (code,
   comments, tests, `docs/ERROR_DESIGN.md`); historical status reports left alone.
9. **flake meta polish** — `meta.homepage` + `meta.platforms` on both packages;
   `vendorHash` was ALREADY extracted (TODO premise stale again).
10. **Brittle `gemini-2.5-flash` test** — now picks whatever model the catalog
    lists under the normalized provider; asserts the alias, not a hardcoded ID.
11. **`docs/DUPLICATION_POLICY.md` refreshed** — re-ran art-dupl over the daemon
    code: extracted 3 actionable clones (`ReviewsDir/FilePermission` exported and
    shared with doctor; `parseConfigFlag`; `openConfiguredPipeline`+`closeStore`),
    accepted one intentional 6-line `once`/`run` pair, documented all of it.

### SDK & docs polish (all 4 items closed)

12. **godoc examples** — `ExampleWrap`/`ExampleIsRetryable` (`pkg/errors`) and
    `ExampleConfig_validate` (`pkg/vision`), all with `// Output:` verified.
13. **`internal/cli` tests** — `NewAgent` error paths (sentinel + `temperature=%.2f`
    context asserted), happy path, and `RequireArgc` exit-code via subprocess
    re-exec (`//nolint:gosec` + `exec.CommandContext` to satisfy the linters).
14. **CHANGELOG → ERROR_DESIGN.md cross-link** — the v0.1.0 error-system entry now
    points at the doc and notes the 16-kind taxonomy.
15. **Mock field priority in AGENTS.md** — verified ALREADY documented at :174;
    no change needed (TODO premise stale).

### Release mechanics

16. **Tag anomalies resolved as far as possible autonomously** — verified
    `v0.3.0` is proxy-burned (proxy serves its `.info` at `d5dda4b`); shipped
    `retract v0.3.0` (with reason comment) in go.mod as the consumer-visible
    remedy. `go list -m -versions` no longer offers v0.3.0. Remote tag deletion
    remains gated on explicit user approval (destructive, cosmetic-only benefit).
17. **Release cut — twice, see (d)** — v0.6.0 AND v0.6.1 tagged+pushed,
    proxy-indexed, GitHub Releases created, consumer-side `go get` verified in a
    clean module (SDK builds with AND without `GOEXPERIMENT=jsonv2` — the
    experiment is only needed for in-repo daemon builds).

### Activation groundwork

18. **SystemNix input bumped** — lock updated `de7c4c6` → `dcd50a0` (includes the
    NixOS module + all of this session's work). Left uncommitted because the
    SystemNix tree carries the user's unrelated WIP.

### Also done along the way (not on the TODO)

- `/visionreviewd` binary output anchored in `.gitignore` (a stray build artifact
  was sitting untracked in the repo root).
- Full verification matrix executed green: build/vet/gofmt, `go test -race`,
  jsonv2 build/vet/test, `go mod tidy -diff` empty, `go mod verify`,
  `nix build .` + `.#visionreviewd`, `nix flake check`, `nix run .#lint/.#test`.
- AGENTS.md updated with durable learnings (UpdatedAt helper, cancelled-pass
  semantics, flake-check gotchas, version-var policy).
- dprint formatting drift on 4 markdown files fixed (was failing BuildFlow).

---

## b) PARTIALLY DONE

1. **"Green badge on the release"** — v0.6.1's tag commit (1bc1523) IS CI-green,
   but the module content published as v0.6.1 predates the bookkeeping commit
   (dcd50a0): its `version` vars still say `"0.6.0"` and it has no `[0.6.1]`
   CHANGELOG section (those landed on master after the tag). Cosmetic mismatch,
   nix-built binaries are unaffected (ldflags inject the real version).
   ← routed 2026-08-18 to ROADMAP open question 5 (release presentation
   policy)
2. ~~**pkg.go.dev refresh** — the fetch trigger returned 404 (normal propagation
   lag); re-check later.~~ done — v0.6.1 published on pkg.go.dev (verified
   2026-08-18)
3. **govulncheck follow-up** — 5 stdlib vulns (GO-2026-6218/6090/6088/5972/5026)
   all fixed in go1.26.6; local+nix toolchain is 1.26.5. Recorded as a TODO item
   (bump when nixpkgs ships 1.26.6) rather than desyncing Go/nix toolchains.

---

## c) NOT STARTED (host-side, cannot be done from this repo)

1. **Enable on a host** — import `nixosModules.visionreviewd`, wire
   `/etc/visionreviewd/config.json`, optional llama-server (~10 GB pull), gate
   with `visionreviewd doctor`.
2. **Real-model smoke test** — `visionreviewd once` against a real llama-server;
   sanity-check review markdown + INDEX; tune caption-tuned prompts.
3. **Point at real projects** — DiscordSync goldens first, then `discover`.
4. ~~**SystemNix lock commit** — bump is applied to the lock but sits uncommitted
   in a tree with the user's WIP.~~ done — committed; lock pins `dcd50a0`
   (verified 2026-08-18, SystemNix working tree no longer carries flake.lock
   changes)

---

## d) TOTALLY FUCKED UP (and recovered)

1. **Tagged v0.6.0 before CI proved green.** I ran the full local verification
   matrix, tagged, and pushed — then CI failed with TWO causes my local env
   couldn't catch: (a) CI jobs never got `GOEXPERIMENT=jsonv2` (needed since
   go-cqrs-lite joined; my shell has it via `go env`, so everything passed
   locally — a blind spot I should have caught by reading the workflow when the
   daemon dependency landed), and (b) `golangci-lint-action` v6 **cannot run
   golangci-lint v2 at all** and silently resolved `version: latest` to a stale
   v1.64.8 binary built with go1.24. Two fix commits later (f76a14d GOEXPERIMENT,
   1bc1523 action v7.0.1 + pinned v2.12.2), CI went green. Per tag immutability,
   the fix ships as **v0.6.1**; v0.6.0 keeps a red badge forever (release note
   says so).
2. **v0.6.1 tag-push race.** The combined commit+tag+push ran while BuildFlow
   pre-commit FAILED (dprint exit 14 on the nix fallback, plus real formatting
   drift). Result: the tag was pushed pointing at 1bc1523 (the intended commit —
   lucky) while the CHANGELOG/version-vars commit failed and landed later as
   dcd50a0. The proxy burned v0.6.1@1bc1523 within seconds. Net effect = the
   content mismatch described in (b). Lesson: never bundle tag creation with an
   unverified commit in one shell line.
3. **Interrupted question call** — my end-of-session approval request (tag
   deletion et al.) was interrupted; decisions remain open, listed in (g).

---

## e) WHAT WE SHOULD IMPROVE

1. **CI-parity for local verification** — local `go env` flags (jsonv2) masked a
   CI-only failure for days. Consider a `nix run .#ci` app or making the
   devShell's lack of the experiment loud. At minimum: read the workflow file
   when a new build-env requirement lands.
2. **Never tag ahead of a green run of the exact commit** — `gh run watch` before
   tagging costs 4 minutes and would have saved v0.6.0's permanent red badge.
3. **Separate tag pushes from commits** — two commands, verify the commit hash
   first. The race in (d).2 was self-inflicted.
4. **BuildFlow dprint flake** — `exit status 14` via the `nix run nixpkgs#dprint`
   fallback is flaky (eval-cache contention); also dprint drift accumulates
   because nothing else checks markdown formatting. Run `nix fmt`/dprint in the
   pre-commit habit, or let treefmt own markdown.
5. **Release-content sync** — version vars + CHANGELOG section must land BEFORE
   the tag; the go-release skill's Phase 4.3 says exactly this and I compressed it.
6. **TODO premises rot** — three items were already done or misdescribed
   (vendorHash extracted, exhaustruct quiet, AGENTS mock note present). The
   docs-health VERIFY pass keeps paying off; keep citing evidence per item.
7. **GitHub "Latest" pointer** — all releases are marked prerelease, so the
   Latest badge still sits on v0.2.0 (July). Decide the 0.x presentation policy.

---

## f) NEXT UP (priority order)

1. User decisions from (g): tag deletion, v0.6.1 mismatch handling, Latest pointer.
   ← still open — consolidated into ROADMAP open question 5 (2026-08-18);
   tag deletion also in TODO_LIST
2. ~~Commit the SystemNix lock bump (user's tree, user's WIP to reconcile).~~
   done — committed (verified 2026-08-18)
3. Enable visionreviewd on a host (doctor-gated), llama-server optional. ← still
   open (TODO_LIST)
4. Real-model smoke test: `visionreviewd once`, inspect markdown + INDEX, tune prompts. ← still open (TODO_LIST)
5. Point daemon at DiscordSync goldens; add projects via `discover`. ← still open (TODO_LIST)
6. Bump Go toolchain to 1.26.6 when nixpkgs ships it (5 stdlib vulns). ← still
   open (TODO_LIST)
7. ~~Re-trigger pkg.go.dev fetch for v0.6.1; verify docs render.~~ done —
   v0.6.1 rendered (verified 2026-08-18)
8. Reset `version` vars to `0.7.0-dev` at the start of the next cycle (currently
   "0.6.1"). ← still open (TODO_LIST)
9. Consider CI job that builds the SDK WITHOUT jsonv2 to lock in the
   consumer-facing guarantee verified manually this session. ← still open
   (TODO_LIST)
10. Consider extracting vendorHash to `vendorHash.nix` (nix-checker suggestion).
    ← still open (TODO_LIST)
11. Address remaining BuildFlow warnings (go-licenses missing in devShell,
    markdownlint MD013 line lengths in AGENTS.md, assets/ layout). ← still
    open (TODO_LIST "Lint-noise policy decision")
12. ROADMAP open questions (structured hooks payload, 0.x semver policy, erraudit
    gate-vs-advisory) — product decisions, not code. ← still open (ROADMAP
    questions 1–3, plus new question 5)

---

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Remote tag deletion** — delete `v0.3.0` (and/or `v0.2.1`) on origin? Both
   point at `d5dda4b`; the proxy cache and the retract make this purely cosmetic
   git hygiene, and it destroys release-history pointers.
2. **v0.6.1 content mismatch** — the published tag reports version "0.6.0" in
   its binaries and lacks its own CHANGELOG section. Cut a synced v0.6.2, or
   accept and move on (my lean: accept; nix builds inject the right version)?
3. ~~**GitHub "Latest" release** — keep marking everything prerelease (v0.2.0
   holds the Latest badge), or promote v0.6.1 to a full release?~~ routed —
   ROADMAP open question 5 (release presentation policy, added 2026-08-18)
