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
      [`docs/visionreviewd-systemnix.md`](docs/visionreviewd-systemnix.md).
      Gate with `visionreviewd doctor`.
- [ ] **Full DiscordSync watch: 216 views** — fold the full
      `internal/web/testdata/visual/*.png` glob (from `discover`) into a
      durable config and pick an interval. At ~20–30 s/view on CPU a full
      first pass is ~1.5–2 h; skip-seen makes later passes incremental.
      User call on cadence.
- [ ] **llama `--image-min-tokens 1024` consideration** — llama-server logs
      the upstream Qwen-VL grounding warning; evaluate whether raising the
      image token budget improves a2ui fidelity on dense screenshots
      (requires restarting llama-server with the flag).

## json/v2 flapping defense

- [ ] **External root fix (user action)** — add `encoding/json` to
      go-auto-upgrade's exclusion list or disable its json rule; it has broken
      compilation 4 documented times (07-27, 07-28, 08-02, 08-18). The repo
      side is fully guarded (depguard + CI grep, probe-verified).

## Release mechanics

- [ ] **Delete the ghost tags (needs explicit approval)** — `v0.2.1` and
      `v0.3.0` both point at `d5dda4b`. `v0.3.0` is already burned into
      proxy.golang.org (verified 2026-08-17), so deletion only cleans the git
      remote; `retract v0.3.0` in go.mod remains the consumer-visible remedy.
      Pair with the release presentation decision (promote v0.6.1 / cut a
      synced v0.6.2 with the 2026-08-18 work).
- [ ] **Reset `version` vars to `0.7.0-dev` when the next cycle opens** —
      both `cmd/vision` and `cmd/visionreviewd` currently say `"0.6.1"`
      (per the AGENTS.md version-var convention). Do this right after the
      release presentation decision, not before.
- [ ] **Bump the Go toolchain when nixpkgs ships 1.26.6** — govulncheck
      reports 5 stdlib vulnerabilities (GO-2026-6218, GO-2026-6090,
      GO-2026-6088, GO-2026-5972, GO-2026-5026), all fixed in go1.26.6.
      Probed 2026-08-18: locked AND unstable nixpkgs still ship 1.26.5
      (unstable also has 1.27rc3); bumping `go.mod` alone would desync Go-
      and nix-built binaries. Re-probe on nixpkgs bumps.

> Product questions that gate future work live in
> [ROADMAP.md](ROADMAP.md#open-questions) (structured hooks payload, semver
> policy for 0.x, `erraudit` as gate vs advisory, release presentation
> policy).
