# TODO List

Short- and mid-term actionable tasks. Each item is bounded and scoped.
For long-term ideas and raw direction, see [ROADMAP.md](ROADMAP.md).
For current feature inventory, see [FEATURES.md](FEATURES.md).

> **Discipline:** When a task is completed, **remove it from this file** and
> record it under `[Unreleased]` in [CHANGELOG.md](CHANGELOG.md). This file is
> for open work only — no `[x]` checkboxes, no "Previously Completed" sections.

---

## visionreviewd activation (next steps)

- [ ] **Enable on a host via SystemNix (user action, needs sudo)** — the
      SystemNix lock pins `dcd50a0` (committed, verified 2026-08-18). Import
      `nixosModules.visionreviewd`, set
      `configFile = "/etc/visionreviewd/config.json"` (template:
      `docs/visionreviewd-config.example.json`, worked example:
      `docs/activation/`), optionally `llamaServer.enable = true` (first start
      waits out the model load via the `/health` readiness probe). Steps:
      [`docs/visionreviewd-systemnix.md`](docs/visionreviewd-systemnix.md)
      (now includes the journal backup ritual). Gate with `visionreviewd
      doctor` — verified 2026-09-07 against the example config: 5 checks,
      actionable per-check output, exit 1 on failure.

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
