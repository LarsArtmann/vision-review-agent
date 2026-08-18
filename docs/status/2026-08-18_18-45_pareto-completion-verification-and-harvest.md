# Status: Pareto Plan Complete — Unwrap Verified, Matrix Green, Harvest Done

**Date:** 2026-08-18 ~18:45
**Session:** Resumed from the 15:05 halt report
(`2026-08-18_15-05_pareto-execution-trust-rails-first-real-reviews.md`)
after the user's go-ahead; finished the plan end to end.

That report remains the detailed a–g ledger for the whole effort; this file
closes it out.

---

## Resumed at the stop point and finished

1. **`*`-unwrap verified.** The halt left `generate.go`/`component.go`/
   `generate_bdd_test.go` uncompiled. Build was green; one BDD assertion was
   wrong (root containers legitimately have no props — blanket `BeEmpty`
   check fixed to assert prop survival on the wrapped components). Lint
   fallout in the three files + `bench_test.go` fixed (forcetypeassert, wsl,
   perfsprint, prealloc, golines/gofumpt).
2. **Real-model validation: ALL LINES VALID.** Rebuilt CLI → llama run on
   `Dashboard--light--desktop.png` → 22-component surface, 2 messages,
   official-schema validator passes both lines. First attempt hit a _different_
   quirk (dangling child refs `archive-card-2..6` — correctly rejected with a
   precise error); re-run succeeded.
3. **Real comparison + events sanity.** `visionreviewd compare` light→dark
   Dashboard through the real model: accurate structured comparison
   (improved/worse/unchanged/verdict + score), `view.compared` event on the
   journal (17 events total, all kinds present). llama-server had died
   mid-session and was restarted (health-gated).
4. **Full verification matrix green.** Go: build/vet/gofmt/
   `test -race ./...`, `GOEXPERIMENT=jsonv2` regime, `GOEXPERIMENT=none`
   SDK subset, `go mod verify`, `go mod tidy -diff` empty, whole-repo
   `golangci-lint` 0 issues. Nix: `nix run .#test`, `nix run .#lint`,
   `nix build .`, `nix build .#visionreviewd`, `nix flake check` — all pass
   (vendorHash bumped for santhosh-tekuri and extracted to `vendorHash.nix`).
5. **Docs loose ends closed (F21/F33/F42 + M21/M23/M27 + F70).** Coverage
   (89%) + bench numbers in AGENTS.md; builder table in FEATURES.md; README
   `-a2ui` snippet; `docs/A2UI.md` + package README; `docs/BUILDFLOW.md` with
   real `journalctl -k` OOM evidence (global OOM 16:27:58, llama-server the
   hog at ~1 GB RSS + ~4.6 GB swap, 7 session daemons the victims); lint-noise
   policy with working configs (markdownlint + codespell now clean on all 16
   living docs); `no-jsonv2` CI job (SDK subset — the daemon genuinely needs
   jsonv2, correcting the stale "both regimes" AGENTS claim); go-licenses in
   the devShell.
6. **Harvest done.** TODO_LIST trimmed to the true remainder; CHANGELOG
   `[Unreleased]` records the session; plan file annotated (not rewritten);
   activation configs made durable under `docs/activation/`; stale ignored
   artifacts trashed (`coverage.out`, `jscpd-report.json`,
   `.art-dupl-baseline.json`); module graph eyeballed (santhosh-tekuri adds
   only regexp2 + x/text).

## New discoveries this session

- **The "SDK builds under both json regimes" claim was half wrong** — only
  the SDK subset does; `internal/reviewd`/`cmd/visionreviewd` hard-depend on
  jsonv2 via go-cqrs-lite. The new `no-jsonv2` CI job encodes the real
  boundary; AGENTS.md corrected.
- **Dangling-child refs are a recurring quirk** on dense, repetitive
  screenshots (message lists): the model defines N−1 rows and references N.
  Correctly rejected; prompt adjacency rule strengthened ("define all N row
  components and count them"). At temperature 0 this is deterministic per
  image — `Messages_hide_bots--dark--mobile.png` failed 3/3 runs identically
  (the known-failing golden), while `Dashboard--light--desktop.png` validates
  ALL LINES. If it persists, it belongs beside the `*` unwrap in the
  repair-vs-reject policy question.
- **treefmt (flake check) enforces single trailing newline** in `.nix` files
  — a double `\n` fails `nix flake check`.

## Remaining (all user-gated or external)

- M24: ghost tag deletion + release presentation (gates the `0.7.0-dev`
  reset).
- SystemNix host enablement (sudo) — steps in `docs/visionreviewd-systemnix.md`.
- Full 216-view DiscordSync watch + interval decision.
- go-auto-upgrade external config fix; Go 1.26.6 when nixpkgs ships it.

## Questions for the user (unchanged from the 15:05 report)

1. Release presentation: delete ghost tags and/or promote v0.6.1 / cut v0.6.2?
2. Activation depth: SystemNix steps now, or is the user-space stack enough?
3. `*`-unwrap policy: keep as Generate-time repair (current) or reject as
   model error — and is nsfwcaption the long-term model?
4. (new) llama-server on 8390: keep running or kill after this session?

llama-server was left RUNNING (restartable via the command in
`docs/activation/README.md`).

---

## Addendum: decisions executed (post-report)

The user answered the gated questions: commit in slices, **cut v0.6.2 +
delete ghost tags**, user-space activation is sufficient, **keep the
`*`-unwrap repair**, keep llama-server running. Executed:

- 5 commits (conformance+guards / builders+decompile+generate /
  daemon+replay+activation / docs+hygiene / release prep), pushed; CI green
  on the release commit including the json/v2 grep guard's and the
  `no-jsonv2` job's first real runs.
- **v0.6.2 tagged** (`ac2172f`, annotated), pushed, proxy verified
  (`v0.6.2.info` serves the right hash), `go get` in a clean module works
  (under the default Go regime), GitHub prerelease published.
- **Ghost tags `v0.2.1`/`v0.3.0` deleted** (both dereferenced the known
  ghost commit `d5dda4b`; verified before deletion). Version list is now
  clean with v0.6.2 latest.
- Cycle opened: `version` vars reset to `0.7.0-dev` in both binaries.
- dprint excludes `**/testdata/**` — the pinned official schemas must stay
  byte-identical to upstream (hash-verified against `29b715fa` after the
  hook first flagged them).
