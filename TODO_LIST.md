# TODO List

Short- and mid-term actionable tasks. Each item is bounded and scoped.
For long-term ideas and raw direction, see [ROADMAP.md](ROADMAP.md).
For current feature inventory, see [FEATURES.md](FEATURES.md).

> **Discipline:** When a task is completed, **remove it from this file** and
> record it under `[Unreleased]` in [CHANGELOG.md](CHANGELOG.md). This file is
> for open work only — no `[x]` checkboxes, no "Previously Completed" sections.

---

## pkg/vision/a2ui — Go 1.27 `encoding/json/v2` wire regression (found 2026-09-22)

`go.mod` moved to Go 1.27.1, where `encoding/json/v2` exists natively (no
GOEXPERIMENT) with different defaults than the 1.26 experiment. Two a2ui tests
now fail on master (verified at committed HEAD in a clean worktree, so no
local-change blame):

- [ ] **Restore deterministic wire output** — `Component.MarshalJSON` and
      `UpdateDataModel.MarshalJSON` marshal `map[string]any`; under 1.27 the
      key order is nondeterministic (5 runs → 4 orderings). Protocol output
      must be byte-stable: emit map keys sorted (ordered encoding path), then
      re-pin `ExampleCompile`'s `// Output:` block.
- [ ] **`omitempty` ignored on scalars** — `CreateSurface.MarshalJSON` emits
      `"sendDataModel":false` despite `json:"sendDataModel,omitempty"`, which
      breaks `TestMessageWireShape/createSurface{,_with_theme}`. Switch the
      a2ui marshal tags to `omitzero` (honored by v1 and v2) or gate the field
      in code, and confirm `theme,omitempty` behaves identically.
- [ ] **Lint fallout of the new `.golangci.yaml` rules (uncommitted change)** —
      `wsl_v5` ×3 (markdown.go, pipeline_bdd_test.go), `gocognit` ×1
      (golden_journal_test.go), `tagliatelle` ×3 on `sourceURLs`/`sourceURL`
      wire tags (config.go, events.go — URL is not a "word", add the
      documented `//nolint:tagliatelle` with the openaicompat precedent), and
      `exhaustruct_v5` ×4 (Pipeline/Captured missing recently added SourceURL
      fields — populate or nolint; bbolt.Options literals ×2 are deliberate
      zero-value opens).

---

## Website fleet — monitoring & re-review (2026-09-19 sweep residue)

Harvested from the 2026-09-19 fleet follow-up execution
(`emeet-pixyd/docs/planning/2026-09-19_15-33_website-vision-review-followup-pareto-plan.md`,
status: `emeet-pixyd/docs/status/2026-09-19_20-14_website-fleet-followup-execution.md`).

- [ ] **KNOWN_BROKEN escalation timer** — `site-monitor.sh` tracks
      cmdguard.lars.software / typespec-asyncapi.lars.software as KNOWN
      (no alert, ever). Escalate (alert) once an entry has been KNOWN for
      > 7 days so "known" cannot mean "forever".
- [ ] **Off-machine monitor vantage** — site-monitor runs on the box whose
      /data disk is failing and alerts via notify-send (desktop-only). Add
      one external check (uptime push or second host) so fleet-down is
      visible when this machine is down.
- [ ] **home-manager module for both timers** — `site-monitor.timer` +
      `fleet-review.timer` are installed as user symlinks in
      `timers.target.wants/`; a home-manager/systemd unit definition makes
      the install canonical and wipe-proof.
- [ ] **`site-monitor.sh --json` consumer** — the flag exists; wire a
      waybar/status widget to it.
- [ ] **Score-trend digest in fleet-review log** — after each cycle, append
      INDEX deltas vs the previous cycle (the data is in the reviews dir;
      review-fleet.sh currently prints per-project scores only).
- [ ] **Commit `family-redeploy.sh`** — the /tmp rebuild+redeploy loop went
      14/16 first-try on the family fixes; promote it into `scripts/` with
      the coverage-diff guard from the a11y sweep lesson.

## Website fleet — review depth (Wave 4 tails)

- [ ] **Full-page captures** — shoot script uses viewport-only shots; add
      CDP `captureBeyondViewport` mode, re-capture, re-review (models score
      above-the-fold only today).
- [ ] **Dark-mode capture pass** — chromium `prefers-color-scheme=dark`
      emulation for all 17 homes; triage sites that break in dark mode.
- [ ] **Docs-subpage capture + review** — enumerate each site's
      docs/getting-started pages, capture, review; the a11y audit covered
      home pages only.
- [ ] **Score calibration (needs idle GPU or patience on CPU)** — 3 views ×
      3 samples variance; qwen2.5vl:3b vs 8B on the same views; an
      artifact-suspicion prompt variant; then decide model+prompt and
      document. GPU is currently occupied by an ollama qwen2.5vl:3b (not
      ours); do NOT kill it — ownership unknown.
- [ ] **GPU benchmark llama-cpp-rocm vs CPU** — rocm build exists at
      `/tmp/vra/llama-rocm`; needs a free-GPU window. CPU baseline
      ~30 s/view is on record.

## Website fleet — quality polish

- [ ] **Template-family token convergence — SOURCE DONE 2026-09-20, deploys pending.** All 9 INFO repos now define `--color-on-accent` (per-theme values CONTRAST-MEASURED per repo — go-error-family uses white/white because its dark accent is violet-600; go-atomic-write uses dark-ink/dark-ink on emerald; the rest dark-ink-dark/white-light) and their solid `bg-accent` CTAs use `text-on-accent`. `family-a11y-guard.sh` now reports **0 FAIL / 0 INFO** family-wide (was 68 INFO). REMAINING: `pnpm build` + firebase deploy for the 9 sites (deferred — the box was load 30-88 all night; never build parallel to model inference), then an axe re-run to confirm contrast live.
- [ ] **Template-family divergence report → ADR** — DONE 2026-09-19:
      `docs/activation/template-family-divergence.md` +
      `docs/adr/0001-template-family-sync-over-extract.md` + prototype
      `scripts/family-a11y-guard.sh`. Remaining: use the guard in the next
      family sweep and wire it as a CI check in one pilot repo.
- [ ] **Hero-copy staleness pass** — per-site counts ("123 components") and
      version claims drift; verify each against the repo state.
- [ ] **learnings: docs-subpage heading-order cleanups** — home page fixed;
      docs subpages still have heading-order violations.
- [ ] **templ-components cmd/site vet gate** — pre-existing: site go.mod
      pins 1.26 but code uses go1.27-only jsonv2 API. Decide bump vs guard
      (not caused by the a11y sweep).
- [ ] **emeet-pixyd online-state screenshots** — TODO #129 in that repo;
      blocked on PIXY hardware being wired.
- [ ] **typespec-asyncapi `website/video/` cleanup** — unused capture
      artifacts committed in the repo.
- [ ] **CSP re-check after og/meta changes** — the report-only CSPs should
      not trip on the new og:image/meta tags; verify no violations were
      observed post-deploy.
- [ ] **auditlog site canonical-URL audit** — decide whether
      go-workflow-auditlog gets the same canonical/sitemap treatment as the
      other 16.
- [ ] **learnings: CI decision (Lars)** — the only workflow
      (`ci.yml`, Go 1.21 "CI/CD Pipeline") is `disabled_manually` in
      Actions (checked 2026-09-20; that's why `gh run list` shows only
      Dependabot). A "manual CI run" is impossible without re-enabling an
      owner-disabled workflow. Also: it tests the Go extractor + docker
      publish, NOT the website — the site has no CI, deploys are local
      firebase. Options: leave disabled, re-enable + prove green, or
      delete the stale workflow.
- [ ] **Dark-mode coverage gaps** — cmdguard is dark-first with a manual
      toggle (no `prefers-color-scheme` media query): system-dark users get
      the default (fine), but system-light users also get dark; consider a
      media-query default. learnings now respects `prefersColorScheme`.
      Remaining views are captured; triage lives with the monthly cycle.

## Blocked on Lars (console / DNS / sudo / decisions)

- [ ] cmdguard.lars.software: attach domain in Firebase console (site works
      at cmdguard.web.app; steps in `docs/activation/site-fleet-ops.md` §1).
- [ ] typespec-asyncapi.lars.software: Namecheap CNAME →
      typespec-asyncapi.web.app, then attach (steps in same doc).
- [ ] `sudo smartctl -a /dev/nvme1n1` → /data verdict; then decide
      replace-vs-selective-migration (VL model ≈ 9 GB → root NVMe would cut
      the 13-min cold reload to seconds).
- [ ] Identify the owner of the running ollama qwen2.5vl:3b GPU workload
      (36+ min at 100%); keep-or-delete decision (ROADMAP seed).
- [ ] Push policy for local commits (vision-review-agent, learnings, fleet
      sites — all currently local-only).

---

## visionreviewd activation (next steps)

- [ ] **Finish the evo-x2 SystemNix enablement (user action: push + deploy)** —
      SystemNix wiring is DONE (2026-09-22 session, commits `c2010e92` +
      `6349307e` there): input re-added and locked (`c8ca4b5`), wrapper module
      `modules/nixos/services/visionreviewd.nix` imports the upstream module
      directly, evo-x2 enables the service with `/etc/visionreviewd/config.json`
      generated from a 17-site list (baseUrl = llama-vlm's captioner :8128,
      NOT `llamaServer.enable` — that stays false on this host; fresh journal
      under `/var/lib/visionreviewd`). Remaining (owner, needs sudo/push):
      1. push this repo (flake now builds again: `buildGoModule` overrides
         `go = pkgs.go_1_27` — the 1.27.1 go.mod floor had broken all nix
         builds), 2. in SystemNix: `nix flake lock --update-input
         vision-review-agent` (mandatory — the current lock predates the go
         fix and its build fails), 3. `nix run .#deploy`, 4. verify:
         `systemctl status visionreviewd`, `sudo visionreviewd doctor -config
         /etc/visionreviewd/config.json` (pre-deploy gate was 20/21 green —
         the model-id check flips green once the new `--alias
         nsfwcaption-qwen3-vl-8b-v3` on llama-vlm-cap deploys), then watch
         `journalctl -u visionreviewd -f` and the first pass land under
         `/var/lib/visionreviewd/reviews`. Journal backup ritual:
         [`docs/visionreviewd-systemnix.md`](docs/visionreviewd-systemnix.md).

## DiscordSync full watch (user decision: cadence)

- [ ] **Fold the full visual glob into a durable config and pick an interval**
      — `discover /home/lars/projects/DiscordSync` already emits the full
      glob; measured 2026-09-07: **220 PNGs** in
      `internal/web/testdata/visual/`. Durable config entry:
      `json
      "projects": { "discordsync":
        ["/home/lars/projects/DiscordSync/internal/web/testdata/visual/*.png"] }`
      Duration estimate from the measured baseline
      (`docs/status/2026-09-07_22-15_perf-baseline-m14.md`): local scan+fold
      is ~6 ms for 220 views (PassSteady 2.9 ms per 100); the model dominates
      at ~20–30 s/view on CPU → **first full pass ≈ 1.2–1.8 h**. Skip-seen
      makes later passes incremental (model only runs on changed hashes, scan
      cost negligible). Cadence options: **10 m** (current; fine — scan is
      ~free, models only fire on changes), **1 h** (recommended default for a
      220-view corpus), **6 h / nightly** (if llama-server CPU/GPU is shared
      with other work). User call.

## llama `--image-min-tokens 1024` eval (user action: llama restart)

- [ ] **Evaluate raising the image token budget on dense screenshots** —
      llama-server logs the upstream Qwen-VL grounding warning. Plan (do NOT
      run against the live unit until a restart is scheduled):
      1. Pick 3 dense goldens (message-list views from the DiscordSync
      corpus).
      2. Baseline: current unit, `visionreviewd once` into a scratch
      `dataDir`; record per-view score + review markdown.
      3. Variant: `systemctl edit llama-vision-server` with
      `ExecStart=` override adding `--image-min-tokens 1024` (the module's
      ExecStart in `nixos/visionreviewd.nix:123` is hardcoded; use a
      transient override for the A/B, promote to a module option only if
      the eval wins), restart, repeat step 2 into a second scratch dir.
      4. Compare: score deltas, review specificity, and (if exercising a2ui)
      compiled-surface component counts + validation failures. Adopt the
      flag permanently only on a clear fidelity win at acceptable
      throughput cost; otherwise close with the measurement recorded.

## json/v2 flapping defense (user decision)

- [ ] **External root fix: exclude the jsonv1tov2 migrator for this repo**
      — root cause located (2026-09-07, verified in go-auto-upgrade source):
      BuildFlow's go-auto-upgrade step applies its `jsonv1tov2` migrator to
      every repo under `~/projects`; this repo needs `encoding/json` (v1 API)
      because the jsonv2 experiment swaps the implementation, not the import
      path — see AGENTS.md "Dual json v1+v2 support". The 4 breakages:
      2026-07-27, 07-28, 08-02, 08-18. The tool reads a per-project
      `.go-auto-upgrade.json` (`{"exclude": ["jsonv1tov2"]}`, schema:
      `include`/`exclude` arrays of migrator names — `cmd/go-auto-upgrade/
      config.go`). **Proposal: commit that file at this repo root.** The repo
      side is already guarded (depguard + CI regime jobs), so this is
      defense-in-depth at the source.
- [ ] **go-cqrs-lite asks (filed 2026-09-07)** — #22 (bbolt
      `OpenWith(ReadOnly)` always fails) and #23 (missing `serializableEvent`
      golden/contract test) are filed; a per-module CHANGELOG request was
      evaluated and dropped: the main CHANGELOG already tracks submodule
      releases centrally, so the ask adds no value. Nothing to do here
      unless the user wants different handling.

## Release mechanics

- [ ] **go-cqrs-lite release cadence (gates M9; user call)** — upstream
      master carries v5-prep commits; the daemon pins v4-era submodules via
      pseudo-compatible tags. Decide (see ROADMAP open question #5): release
      upstream now and re-bump the consumer, or wait for a consumer-driven
      trigger. Blocked on the cadence answer — do not tag the sibling repo
      without it.
- [ ] **Bump-PR body template (reusable)** — for future dependency-bump PRs
      (next up: fantasy v0.41.1→v0.43.1, catwalk, testify/gomega minors —
      `nix run .#dep-drift` list):
      ``markdown
      ## Summary
      - Bump <dep> <from> → <to> (<why: CVE / feature / drift>)
      ## Verification
      - [ ] `nix run .#verify-bump` (build/vet/gofmt/lint/race/tidy/verify)
      - [ ] `nix build .#visionreviewd` (vendorHash in sync — if stale:
            `nix run .#update-vendor-hash`)
      - [ ] GOEXPERIMENT=jsonv2 full suite + no-jsonv2 SDK subset
      - [ ] govulncheck 0 reachable``
- [ ] **0.x `--latest` / prerelease policy** (ROADMAP open question #5) —
      decide how releases are presented on GitHub (v0.7.0 currently holds
      `--latest` correctly).

> Product questions that gate future work live in
> [ROADMAP.md](ROADMAP.md#open-questions) (structured hooks payload, semver
> policy for 0.x, `erraudit` as gate vs advisory, release presentation
> policy).
