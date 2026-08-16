# TODO List

Short- and mid-term actionable tasks. Each item is bounded and scoped.
For long-term ideas and raw direction, see [ROADMAP.md](ROADMAP.md).
For current feature inventory, see [FEATURES.md](FEATURES.md).

> **Discipline:** When a task is completed, **remove it from this file** and
> record it under `[Unreleased]` in [CHANGELOG.md](CHANGELOG.md). This file is
> for open work only — no `[x]` checkboxes, no "Previously Completed" sections.

---

## visionreviewd activation (post-build)

- [ ] **Finish the v0.6.0 release** — CHANGELOG, version vars, `retract
      v0.3.0`, and the full verification matrix are done on the working tree;
      what remains is commit → push → wait for green CI (the lint job's
      `golangci-lint config verify` step is the proof the wrapcheck fix works
      with `@latest`) → annotated tag → push tag → consumer-side `go get`
      verification → GitHub Release (`--prerelease` per 0.x policy).
- [ ] **Bump the SystemNix `vision-review-agent` input** — both repos are
      pushed, but the SystemNix lock still pins `de7c4c6`, which predates the
      NixOS module (`b2d9c0c`), so the lazy wrapper stays inert. Run
      `nix flake lock --update-input vision-review-agent` in SystemNix **after**
      the v0.6.0 tag is pushed. (Steps: `docs/visionreviewd-systemnix.md`.)
- [ ] **Enable on a host** — import `nixosModules.visionreviewd`, set
      `configFile = "/etc/visionreviewd/config.json"` (template:
      `docs/visionreviewd-config.example.json`), optionally
      `llamaServer.enable = true` (~9–10 GB model pull on first start). Gate
      with `visionreviewd doctor`. The module now has CI eval coverage
      (enabled + disabled) via the flake checks.
- [ ] **Real-model smoke test** — all E2E coverage uses the fake
      OpenAI-compatible server (`internal/reviewd/fakeserver_test.go`); run
      `visionreviewd once` against a real llama-server, sanity-check one
      review markdown + INDEX, and tune the caption-tuned prompts.
- [ ] **Point the daemon at real projects** — start with DiscordSync goldens,
      then add more projects via `visionreviewd discover`.

## Release mechanics

- [ ] **Delete the duplicate remote tags (needs explicit approval)** —
      `v0.2.1` and `v0.3.0` both point at `d5dda4b`. `v0.3.0` is already
      burned into proxy.golang.org (verified 2026-08-17: the proxy serves its
      `.info`), so deletion only cleans the git remote; go.mod in v0.6.0
      carries `retract v0.3.0` as the consumer-visible remedy. Deleting
      remote tags is destructive — get the user's explicit approval first.
- [ ] **Bump the Go toolchain when nixpkgs ships 1.26.6** — govulncheck
      (BuildFlow pre-commit, 2026-08-17) reports 5 stdlib vulnerabilities
      (GO-2026-6218, GO-2026-6090, GO-2026-6088, GO-2026-5972, GO-2026-5026),
      all fixed in go1.26.6. The local/nix toolchain is 1.26.5, so bumping
      `go.mod` now would desync Go- and nix-built binaries; revisit when
      `pkgs.go_1_26` advances.

> Product questions that gate future work live in
> [ROADMAP.md](ROADMAP.md#open-questions) (structured hooks payload, semver
> policy for 0.x, `erraudit` as gate vs advisory).
