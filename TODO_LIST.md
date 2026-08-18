# TODO List

Short- and mid-term actionable tasks. Each item is bounded and scoped.
For long-term ideas and raw direction, see [ROADMAP.md](ROADMAP.md).
For current feature inventory, see [FEATURES.md](FEATURES.md).

> **Discipline:** When a task is completed, **remove it from this file** and
> record it under `[Unreleased]` in [CHANGELOG.md](CHANGELOG.md). This file is
> for open work only — no `[x]` checkboxes, no "Previously Completed" sections.

---

## A2UI verification & hardening

From the 2026-08-18 audit
(`docs/status/2026-08-18_12-55_a2ui-subpackage-brutal-self-review.md`).
The nix half of the verification matrix has since gone green
(`2026-08-18_13-21`); this is what actually remains.

- [ ] **Official-schema conformance test** — run generated messages through
      the official `v0_9_1/json/server_to_client.json` +
      `common_types.json` schemas with a draft-2020-12 validator
      (`kaptinlin/jsonschema` is already an indirect dep through fantasy).
      Until then the docs claim "implements the v0.9.1 message shapes"
      (reworded 2026-08-18), not schema-validated.
- [ ] **Decide + document `"value": null` semantics** — `UpdateDataModel`
      cannot distinguish explicit null from an omitted value (both decode to
      `Remove: true`; the spec says omitted = remove, explicit null is a
      legal write). Presence flag or documented limitation.
- [ ] **Decide + document unknown-key handling** — `UnmarshalMessage`
      silently ignores unknown top-level keys; the official schema sets
      `additionalProperties: false`. Reject like the schema, or document the
      leniency.
- [ ] **Run the flake's own gates over the a2ui surface** — `nix run .#test`
      and `nix run .#lint` (the pinned linter may differ from the system
      v2.12.2 that verified the depguard guard); record
      `go test -cover ./pkg/vision/a2ui/...`.
- [ ] **art-dupl scan over `pkg/vision/a2ui`** — the 9 near-identical
      builders were never scanned; update
      [`docs/DUPLICATION_POLICY.md`](docs/DUPLICATION_POLICY.md) with the
      judgment calls (currently 0 a2ui mentions, verified 2026-08-18).
- [ ] **Fuzz the codec entry points** — `UnmarshalMessage`,
      `UnmarshalJSONL`, `Component.UnmarshalJSON`, `ChildList.UnmarshalJSON`.
- [ ] **Pin the prompt catalog** — test asserting `catalogSignatures()`
      covers exactly the 19 basic-catalog kinds (or generate the block from
      `catalog.json` via go:generate), and assert
      `SchemaName == "SurfaceSpec"` in the BDD fake.
- [ ] **Benchmarks + godoc examples** — `BenchmarkMarshalJSONL`,
      `BenchmarkValidate`, `BenchmarkUnmarshalMessage`; `Example_*`
      functions (testableexamples verifies output).
- [ ] **Finish the builder set** — constructors for the remaining 10
      basic-catalog kinds (CheckBox, ChoicePicker, DateTimeInput, Slider,
      Tabs, TextField, List, Modal, AudioPlayer, Video) and document which
      kinds lack builders.
- [ ] **`GenerateOptions` completeness** — `Theme` passthrough to the
      createSurface message; `DataModel` seed merged with the model's output.
- [ ] **`Decompile(messages) → SurfaceSpec`** — close the `Compile`
      asymmetry (enables edit round-trips and diffing).
- [ ] **Narrow the exhaustruct exclusion** — the whole-package
      `pkg/vision/a2ui\..*$` exclude hides future genuine exhaustiveness
      bugs; scope it back to the wire/message types (or accept, but decide
      explicitly).
- [ ] **CLI exposure** — `cmd/vision -a2ui mockup.png` printing JSONL.
- [ ] **Glossary sweep** — `docs/DOMAIN_LANGUAGE.md` has 0 visionreviewd
      terms (viewKey, capture/review/compare events, blob store, pass,
      replay, doctor) and 0 a2ui terms (surface, catalog, adjacency list,
      inference format). Dropped by the 08-17 TODO rebuild; recovered
      2026-08-18.
- [ ] **`docs/A2UI.md` + `pkg/vision/a2ui/README.md`** — format comparison
      (wire vs inference), lifecycle, error taxonomy; package README for
      pkg.go.dev browsers.

## json/v2 flapping defense

The depguard deny rule (lint layer) landed `11d3490`; these close the
non-lint paths. Root cause (go-auto-upgrade's own config) is external.

- [ ] **CI grep regression test** — fail if any `.go` file imports
      `encoding/json/v2` or `encoding/json/jsontext` (depguard only covers
      lint runs; the daemon edits files directly). Decide between a CI grep,
      a Go test, or a pre-commit hook — one mechanism, not three.
- [ ] **External root fix (user action)** — add `encoding/json` to
      go-auto-upgrade's exclusion list or disable its json rule; it has
      broken compilation 4 documented times (07-27, 07-28, 08-02, 08-18).
- [ ] **`docs/BUILDFLOW.md`** — one page: known-transient nix OOM retry
      policy + the json/v2 guard explanation, so future sessions don't
      re-triage from scratch.
- [ ] **OOM evidence + buildflow policy** — pull kernel logs for the
      2026-08-18 kill window (the "transient" diagnosis was never proven);
      decide whose knob (`--max-time`/`--default-step-timeout` vs nix
      `--max-jobs`/`-o cores`) and whether the repo flake should cap jobs.

## visionreviewd activation (post-build)

- [ ] **Enable on a host** — the SystemNix lock pins `dcd50a0` (committed,
      verified 2026-08-18). Import `nixosModules.visionreviewd`, set
      `configFile = "/etc/visionreviewd/config.json"` (template:
      `docs/visionreviewd-config.example.json`), optionally
      `llamaServer.enable = true` (~9–10 GB model pull on first start). Gate
      with `visionreviewd doctor`. Module eval is CI-covered (enabled +
      disabled) via the flake checks.
- [ ] **Real-model smoke test** — all E2E coverage uses the fake
      OpenAI-compatible server (`internal/reviewd/fakeserver_test.go`); run
      `visionreviewd once` against a real llama-server, sanity-check one
      review markdown + INDEX, and tune the caption-tuned prompts
      (descriptive→critical contract is unvalidated).
- [ ] **Point the daemon at real projects** — DiscordSync goldens first,
      then add more projects via `visionreviewd discover`.
- [ ] **llama unit readiness gate** — `ExecStartPost` probing `/health` in
      `nixos/visionreviewd.nix` so the daemon's first pass doesn't race the
      ~10 GB model load.

## Test & tooling debt

Dropped by the 08-17 TODO rebuild; recovered 2026-08-18.

- [ ] **`CompareManually` → wipe → `Replay` round-trip test** —
      manual-compare-only streams are handled by replay but never
      round-trip tested end-to-end.
- [ ] **BDD spec for replay** — currently table tests only.
- [ ] **`consumeObjectStream` partial-malformed-object test** — partial
      unmarshal failures during structured streaming are tolerated
      best-effort; untested.
- [ ] **`examples/structured-stream` review** — verify the example matches
      the hardened final-object unmarshal-error contract.
- [ ] **`.golangci.yaml` rationale comments** — the G117/G101 gosec excludes
      still carry no explanation (partial since the catwalk session).

## Release mechanics

- [ ] **Delete the duplicate remote tags (needs explicit approval)** —
      `v0.2.1` and `v0.3.0` both point at `d5dda4b`. `v0.3.0` is already
      burned into proxy.golang.org (verified 2026-08-17: the proxy serves
      its `.info`), so deletion only cleans the git remote; go.mod in v0.6.0
      carries `retract v0.3.0` as the consumer-visible remedy. Destructive —
      get the user's explicit approval first.
- [ ] **Bump the Go toolchain when nixpkgs ships 1.26.6** — govulncheck
      reports 5 stdlib vulnerabilities (GO-2026-6218, GO-2026-6090,
      GO-2026-6088, GO-2026-5972, GO-2026-5026), all fixed in go1.26.6. The
      local/nix toolchain is 1.26.5; bumping `go.mod` now would desync Go-
      and nix-built binaries.
- [ ] **Reset `version` vars to `0.7.0-dev` when the next cycle opens** —
      both `cmd/vision` and `cmd/visionreviewd` currently say `"0.6.1"`
      (per the AGENTS.md version-var convention).
- [ ] **CI job building the SDK WITHOUT jsonv2** — lock in the
      consumer-facing guarantee (verified manually for v0.6.1) that the
      module compiles for plain `go get` consumers.
- [ ] **Extract `vendorHash` to `vendorHash.nix`** — nix-checker suggestion;
      cleaner diffs on dependency bumps.
- [ ] **Lint-noise policy decision** — markdownlint MD013 across AGENTS.md,
      codespell findings in old status docs, `go-structure-linter` assets/
      suggestion, go-licenses missing from the devShell. Fix, configure, or
      ignore-by-design — don't leave ambient warning noise.

> Product questions that gate future work live in
> [ROADMAP.md](ROADMAP.md#open-questions) (structured hooks payload, semver
> policy for 0.x, `erraudit` as gate vs advisory, release presentation
> policy).
