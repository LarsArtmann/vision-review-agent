# TODO List

Short- and mid-term actionable tasks. Each item is bounded and scoped.
For long-term ideas and raw direction, see [ROADMAP.md](ROADMAP.md).
For current feature inventory, see [FEATURES.md](FEATURES.md).

> **Discipline:** When a task is completed, **remove it from this file** and
> record it under `[Unreleased]` in [CHANGELOG.md](CHANGELOG.md). This file is
> for open work only — no `[x]` checkboxes, no "Previously Completed" sections.

---

## CI (critical)

- [ ] **Fix `.golangci.yaml` wrapcheck schema** — `ignoreSigs` and
      `ignore-type-assert-ok` (`.golangci.yaml:239,250`) are rejected by
      `golangci-lint config verify`, and the CI `lint` job runs exactly that
      verify step with `@latest`. CI has been red on every push since
      2026-08-12 (v0.5.0). Rename the keys to the current schema or drop the
      settings, then confirm green CI.
      (Source: `docs/status/2026-08-12_19-00` §D.1/F.1; verified open
      2026-08-16.)
- [ ] **Add CI jobs for the daemon surface** — `nix build .#visionreviewd`,
      a NixOS-module eval check (enabled + disabled), and a smoke run of
      `visionreviewd version` to catch the silent-empty-build class.
      (Source: `docs/status/2026-08-16_22-50` items 21–23.)

## visionreviewd activation (post-build)

- [ ] **Bump the SystemNix `vision-review-agent` input** — both repos are
      pushed, but the SystemNix lock still pins `de7c4c6`, which predates the
      NixOS module (`b2d9c0c`), so the lazy wrapper stays inert. Run
      `nix flake lock --update-input vision-review-agent` in SystemNix.
      (Verified 2026-08-16; steps: `docs/visionreviewd-systemnix.md`.)
- [ ] **Enable on a host** — import `nixosModules.visionreviewd`, set
      `configFile = "/etc/visionreviewd/config.json"` (template:
      `docs/visionreviewd-config.example.json`), optionally
      `llamaServer.enable = true` (~9–10 GB model pull on first start). Gate
      with `visionreviewd doctor`.
- [ ] **Real-model smoke test** — all E2E coverage uses the fake
      OpenAI-compatible server (`internal/reviewd/fakeserver_test.go`); run
      `visionreviewd once` against a real llama-server, sanity-check one
      review markdown + INDEX, and tune the caption-tuned prompts.
- [ ] **Point the daemon at real projects** — start with DiscordSync goldens,
      then add more projects via `visionreviewd discover`.

## Code quality (small, bounded)

- [ ] **doctor stderr injection** — `checkModelEndpoint`'s deferred
      close-error print writes to `os.Stderr` directly, bypassing the
      command's injected writer (`cmd/visionreviewd/commands.go:555`).
- [ ] **Kill `doctorCheckExtra`** — magic constant duplicates knowledge of
      the check count (`cmd/visionreviewd/commands.go:469`); compute it from
      the check list.
- [ ] **INDEX "Updated" column shows `CapturedAt`** — consider `ReviewedAt`
      (`internal/reviewd/pipeline.go:291`, `internal/reviewd/replay.go:235`).
- [ ] **Guard `Pipeline.Pass` on a cancelled context** — verify skip
      semantics are explicit.
- [ ] **exhaustruct excludes for reviewd counter types** — all-zero literals
      like `ReplayResult{...}` are noise; exclude them like
      `pkg/vision.BatchResult` already is.
- [ ] **codespell `unparseable`** — add to ignore-rules or reword
      (`internal/reviewd/replay.go:300,318`, `compare_test.go:126`).
- [ ] **flake meta polish** — add `meta.homepage`/`meta.platforms` to both
      packages; extract the inline `vendorHash` (`flake.nix:49`) for cleaner
      dependency bumps.
- [ ] **Fix brittle `gemini-2.5-flash` test** — hardcodes a catalog model ID
      that breaks when catwalk updates its data
      (`cmd/vision/main_test.go:229`); test the alias, not the model.
- [ ] **Refresh `docs/DUPLICATION_POLICY.md`** — the "current state" claim
      predates `internal/reviewd` (0 mentions); re-run art-dupl and record.

## SDK & docs polish

- [ ] **godoc examples** — testable `Example` functions for `pkg/errors`
      (`errors.AsType[*ModelError]` + `IsRetryable()`) and `pkg/vision`
      (`errors.Is` with enriched sentinel messages).
- [ ] **`internal/cli` has zero tests** — cover `NewAgent`'s error path
      (asserts the `temperature=%.2f` context) and `RequireArgc`.
- [ ] **Cross-link `docs/ERROR_DESIGN.md` from `CHANGELOG.md`** — README and
      AGENTS.md now link it; the CHANGELOG entry predates the document.
- [ ] **Document mock model field priority in AGENTS.md** — the ordering
      (`generateObjectErr` > `generateObjectResponse` > default; streaming
      likewise) lives only in the `mockModel` struct comment.

## Release mechanics

- [ ] **Resolve tag anomalies** — `v0.2.1` **and** `v0.3.0` both point at
      `d5dda4b` and both still exist on origin (verified 2026-08-16 — the
      v0.4.0 release note's claim that `v0.3.0` was deleted does not match
      the current remote state). Deleting remote tags is destructive and
      needs explicit user approval; also consider a `retract` directive for
      the proxy-burned `v0.3.0`.
- [ ] **Cut 0.6.0** — CHANGELOG `[Unreleased]` is loaded with the
      visionreviewd work; fix the CI lint issue first so the release has a
      green badge, and run the consumer-side `go get` verification skipped
      for v0.5.0/v0.5.1.

> Product questions that gate future work live in
> [ROADMAP.md](ROADMAP.md#open-questions) (structured hooks payload, semver
> policy for 0.x, `erraudit` as gate vs advisory).
