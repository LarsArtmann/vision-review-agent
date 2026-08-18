# Status: Pareto Execution — Trust Rails Landed, First Real Reviews Happened

**Date:** 2026-08-18 15:05
**Session:** Execute the Pareto plan
(`docs/planning/2026-08-18_13-42_pareto-a2ui-trust-and-activation.md`) end to
end. Started from clean tree at `c052765`; user halted execution mid-task at
~15:03 ("WAIT AFTER YOU WROTE THE .md file").
**Working tree at halt:** 35 files modified/added, **nothing committed**.
llama-server still running in background (shell `084`, port 8390).

Ground truth changes since plan time: the plan's "1% that delivers 51%" (a
real review of a real screenshot by a real model) **happened** — 8
DiscordSync golden views reviewed with scores 4–7, INDEX rendered, skip-seen
and replay verified. The plan's T2 trust rails are green. T3 is ~80% done.
T4 is untouched.

---

## a) FULLY DONE (verified this session)

### M5 — Official-schema conformance test (F12–F15) ✅

- Pinned the official A2UI v0.9.1 schemas verbatim into
  `pkg/vision/a2ui/testdata/official/` (server_to_client.json,
  common_types.json, basic catalog.json, interactive-button example) from
  upstream `a2ui-project/a2ui` commit `29b715fa` (2026-08-17), with a
  README documenting provenance + refresh procedure.
- **Discovered:** `kaptinlin/jsonschema` v0.9.8 (the TODO's suggested
  validator) **cannot compile the official schemas at all** — its
  exact-number decoder rejects every non-number JSON value decoded into
  `any`, which breaks every string `const` and `enum`. Reproduced
  minimally in a scratch module (compile of a one-const schema fails with
  `unsupported operation`).
- Pivoted to `github.com/santhosh-tekuri/jsonschema/v6` v6.0.3 (new direct
  dep; added to the depguard allow list with a comment explaining why).
- `conformance_test.go`: happy path (spec → Compile → JSONL → validate per
  line), one-of-each-message-kind (incl. remove + dynamic children),
  encode/decode round trip, positive control (official example validates —
  the suite cannot pass vacuously), and a 10-case rejection table with
  decoder-parity flags for the M8-documented leniencies.
- All green; wired into the normal `go test ./...`.

### M6 — json/v2 CI grep guard (F16–F17) ✅

- `.github/workflows/ci.yml`: first step of `build-and-test` now fails on
  any `.go` file importing `encoding/json/v2` / `encoding/json/jsontext`
  (`git grep -nE --untracked '"encoding/json/(v2|jsontext)(/[a-z0-9]+)?"'`).
- Probe-verified in both directions: a temp offending file IS caught; a
  file merely mentioning json/v2 in comments/strings is NOT caught.
- **Lesson learned:** the first version used `git grep` without
  `--untracked` — the probe showed it missed fresh, uncommitted files,
  which is exactly the go-auto-upgrade threat model. Fixed.
- F18 (external root fix) remains a user action, already recorded in
  TODO_LIST.

### M7 — Flake gates (F19–F20) ✅ (F21 partial, see b)

- `nix run .#test`: 0 failures, all 9 packages ok; a2ui coverage **93.4%**
  (measured mid-session, before M13/M15/M22 test additions — number will
  have moved).
- `nix run .#lint`: **0 issues**.

### M8 — Semantics decisions bundle (F22–F25) ✅

- **`"value": null` = write, omitted = remove**: `UpdateDataModel.UnmarshalJSON`
  now detects key presence; round trip preserves the distinction. Tested.
- **Strict envelope, lenient payload**: `UnmarshalMessage` rejects unknown
  top-level keys (schema parity: `additionalProperties: false`), payload
  keys stay lenient for forward compatibility. Tested.
- **exhaustruct narrowed**: whole-package `a2ui\..*$` exclude replaced by
  13 explicitly listed wire/inference types + `ValidationIssue` (each with
  rationale); new a2ui types are no longer auto-excluded.
- All three decisions + rationale documented in AGENTS.md; also fixed the
  long-standing **"19 kinds" misclaim** (the catalog has 18) in two
  AGENTS.md places and updated the package doc claim ("machine-checked" is
  now true).

### M9 — Catalog pin (F26–F27) ✅

- `prompt_pin_test.go` derives the expected kind set from the pinned
  official catalog and asserts `catalogSignatures()` covers exactly it —
  **caught real drift: `Tabs` was missing from the prompt signatures**.
  Added (with true props from the catalog: array of {title, child}).
- BDD fake now asserts `SchemaName == "SurfaceSpec"`.

### M1 — llama readiness gate (F1–F2) ✅

- `nixos/visionreviewd.nix`: llama unit gains `ExecStartPost` curl
  `/health` probe (`--retry 720 --retry-delay 5 --retry-connrefused
  --retry-all-errors`) + `TimeoutStartSec = "infinity"` (first start can
  download ~10 GB; the 90s default would kill it mid-download).
- `nixos-module-enabled` flake check extended to assert the probe exists;
  both enabled/disabled module eval checks green.
- Nix gotcha learned: a list element cannot start with `+` continuation —
  parenthesize.

### M4 — DiscordSync goldens (F10–F11) ✅

- `discover` over `~/projects/DiscordSync` → one glob:
  `internal/web/testdata/visual/*.png` — **216 screenshots** with proper
  viewKey naming.
- `doctor`: globs ok (216 match) + model endpoint ok once llama was up.

### M2/M3 — First real reviews (F3–F9) ✅ via user-space substitute

- `sudo` is blocked in this sandbox → SystemNix host enablement (F4)
  impossible; substituted a plain-user stack:
- Found `llama-server` on the system with the model **already cached**
  (`/data/ai/cache/huggingface`, shared HF_HOME) — started it on
  127.0.0.1:8390 in a background shell. No 10 GB pull needed.
- `doctor` 4/4 green with a one-view config (`/tmp/visionreviewd-first.json`).
- **First real `once`**: 1 captured, 1 reviewed. The Dashboard
  light/desktop review scored 7/10 with accurate, specific, actionable
  content (real observations: bot-disconnected status not actionable,
  inconsistent card padding, footer spacing).
- **Second pass, 8 views** (Dashboard×4, Messages_hide_bots×4): 7 captured,
  1 correctly skipped (skip-seen works), 7 reviewed, INDEX rendered with
  scores 4–7 (full range used). Zero daemon errors.
- Prompt-tuning verdict (F8): the caption-tuned prompts held up — the
  descriptive→critical steering works; the only wart was the duplicated
  "Score: N/10" in the body next to the header bullet. Fixed via
  `StripScoreLines` in view-review rendering (comparisons keep the in-body
  score — their header has no score bullet; documented in code). Verified
  by wiping + `replay`: byte-identical rebuild, single score occurrence.

### M10/M11 — All 18 builders (F28–F32) ✅ (F33 partial, see b)

- 10 new constructors (CheckBox, ChoicePicker, DateTimeInput, Slider,
  Tabs, TextField, List, Modal, AudioPlayer, Video) + `Tab`,
  `ChoicePickerOption` types + enum constants (Choice*, Field*,
  Direction*, Align*) + the missing prop-key constants.
- **Conformance test caught a wrong signature**: catalog requires
  ChoicePicker `value` (DynamicStringList) — added the `value any`
  parameter (literal `[]string` or `Bind`).
- `TestAllBuildersConformToOfficialSchema`: a 30-component surface using
  all 18 kinds passes Validate AND the official schema; per-builder wire
  shape + round-trip table in `builders_test.go` (JSONEq comparison —
  Go slice types widen on decode, struct equality is too strict).

### M12 — Theme + DataModel options (F34–F36) ✅

- `GenerateOptions.Theme` passes through when the model emitted none
  (model theme wins); `GenerateOptions.DataModel` merges as a seed under
  the model's keys (`maps.Copy`, model wins); seeds an omitted data model.
- 4 BDD specs, all green.

### M13 — Codec fuzzing (F37–F40) ✅

- 4 fuzz targets (`FuzzUnmarshalMessage`, `FuzzUnmarshalJSONL`,
  `FuzzComponentUnmarshal`, `FuzzChildListUnmarshal`) with wire-corner
  seeds (two-kind envelopes, CRLF/blank/partial lines, static/dynamic
  child lists). Invariants: no panics, decoded → re-encodes → re-decodes
  stable, JSONL errors always carry a line number.
- Short runs (~35 s total, ~1M execs): zero failures.

### M15 — Decompile (F43–F45) ✅

- `Decompile(messages) → SurfaceSpec`: single createSurface, last
  updateComponents wins, data-model folding with RFC 6901 pointer
  writes/removes (objects + array indices, `~0`/`~1` unescaping).
  Rejects: pre-create updates, double create, deleteSurface, dynamic
  child lists, `sendDataModel` — all via the `ErrDecompile` sentinel.
- Round-trip test (Compile → Decompile → equal), minimal spec, fold order,
  orphan components preserved, 6-case rejection table, edit-loop test
  (compile → decompile → mutate → recompile).
- Lint fallout fixed along the way: `ErrDecompile` wrapped in all dynamic
  errors (err113), `Decompile` split into `decompileMessage` (cyclop 16 →
  under limit).

### M16 — Glossary sweep (F46–F48) ✅

- `docs/DOMAIN_LANGUAGE.md`: visionreviewd vocabulary (11 terms — viewKey,
  golden, capture/review/comparison, pass, blob store, replay, doctor,
  discover, INDEX) + a2ui vocabulary (9 terms — surface, catalog,
  adjacency list, wire/inference format, compile/decompile/generate,
  dynamic value) + 2 bounded-context rows. Event names verified against
  `events.go`; AGENTS.md now links the glossary from the Overview.

### M17 — Duplication scan + policy (F49–F50) ✅

- `art-dupl --type-aware -t 1` over a2ui: 1 actionable clone
  (NewImage/NewAudioPlayer bodies) → extracted `urlDescriptionProps` →
  **0 actionable groups**.
- `docs/DUPLICATION_POLICY.md` gained an a2ui section with judgment calls:
  builder similarity intentional, marshal/unmarshal pairs not merged.

### M18 — Replay round-trip + BDD (F51–F52) ✅

- `internal/reviewd/replay_bdd_test.go`: manual-compare-only stream
  records the event; wipe → `Replay` rebuilds the comparison
  **byte-identically**; replayed INDEX lists the view. Green.

### M19 — Stream tolerance + example review (F53–F54) ✅

- `TestAnalyzeStructuredStreamToleratesMalformedPartials`: malformed
  partial skipped (callback never sees it), valid partial delivered, final
  object wins, text deltas captured. Green.
- `examples/structured-stream` reviewed: already matches the hardened
  final-unmarshal contract (classified errors surface via ExitOnError). No
  change needed.

### M20 — Lint rationale comments (F55–F56) ✅

- G117 (file perms: non-sensitive local artifacts) and G101 (hardcoded
  creds: name-heuristic FPs, gitleaks covers real secrets) each got a
  rationale comment in `.golangci.yaml`.

### M22 — Benchmarks + godoc examples (F60–F62) ✅ (number recording, see b)

- `BenchmarkMarshalJSONL` 129 µs/op, `BenchmarkValidate` 5.9 µs/op,
  `BenchmarkUnmarshalMessage` 223 µs/op (full 3-line, 82-component
  stream), `BenchmarkDecompile` 2.1 µs/op (82-component surface; 1 alloc).
- `ExampleCompile` + `ExampleDecompile` with verified outputs (captured
  from a real run in a scratch module, not hand-written).

---

## b) PARTIALLY DONE (stopped mid-flight)

1. **Model-quirk repair (`*`-unwrap) — THE STOP POINT.** Real-model a2ui
   surfaces (58- and 22-component) failed official-schema validation:
   14–45 components nested their properties under a literal `"*"` key
   (`{"properties": {"*": {"text": ...}}}`). Diagnosis: baked into the
   nsfwcaption fine-tune, NOT prompt confusion — a legend + concrete
   example did not fix it (regeneration still had 14), so I also reworded
   all signatures from `text*` to `text (required, ...)` (clarity win
   regardless) and implemented `unwrapStarProperties` in `generate.go`
   (unwraps when properties is exactly one `"*"` object), `propStar`
   constant, and a BDD spec for it. **NOT done: compile/test after those
   last three edits; NOT re-run against the real model to prove the fix
   end-to-end.** Files touched and unverified: `generate.go`,
   `component.go`, `generate_bdd_test.go`.
2. **F21 (record coverage)**: measured (93.4%) but never written into
   AGENTS/TODO as the task asked.
3. **F33 (FEATURES/README builder table)**: builders done, doc table not.
4. **F42 (README usage snippet for `-a2ui`)**: flag works + verified
   against the real model (stdout JSONL, stderr status, pipeable), README
   not updated.
5. **F3/F4 durability**: activation configs live in `/tmp`
   (`visionreviewd-first.json`, `visionreviewd-six.json`) — volatile; host
   enablement via SystemNix not possible without sudo.
6. **Final verification matrix**: full `go build/vet/gofmt/test -race` was
   green after M15-ish; per-package lint/test runs since then only. Full
   matrix + jsonv2 regime + `go mod tidy -diff` + nix half **not re-run**
   after M18–M22 and the unwrap edit.

## c) NOT STARTED

- **M21** (F57–F59): `docs/A2UI.md` + `pkg/vision/a2ui/README.md`.
- **M23** (F63–F64): `docs/BUILDFLOW.md` + OOM kernel-log evidence.
- **M24** (F65–F66): ghost tag deletion + release presentation (USER).
- **M25** (F67–F68): Go 1.26.6 probe/bump.
- **M26** (F69–F70): `0.7.0-dev` version reset + no-jsonv2 CI job.
- **M27** (F71–F72): `vendorHash.nix` extraction + lint-noise policy table.
- **Docs harvest**: TODO_LIST trimming, CHANGELOG `[Unreleased]` entries,
  plan-file annotation — all deferred to the end per plan; nothing written.
- **Commit**: 35 files staged/unstaged, zero commits this session.

## d) TOTALLY FUCKED UP (honest ledger)

1. **`go build && heredoc-append` chain**: when the build failed (my
   duplicate TextH1 const block), the append silently never ran; I then
   misdiagnosed the missing test file and rewrote it elsewhere. Recovered,
   but self-inflicted. Lesson: never chain edits behind a build.
2. **Guard blind spot**: first json/v2 grep lacked `--untracked` — would
   have missed exactly the fresh-daemon-edited files it exists for. Caught
   only because I probe-verified. (The probe practice saved this one.)
3. **Trusted a TODO's library claim**: "kaptinlin is already an indirect
   dep" — true, but it cannot do the job. ~4 tool calls burned before the
   minimal repro forced the pivot. Should have capability-checked first.
4. **Momentarily weakened a test** (replaced round-trip equality with
   NotEmpty) to silence a trivial unused-variable error; immediately
   restored. Sloppy reflex.
5. **Score-strip overreach**: applied `StripScoreLines` to comparisons too,
   broke the compare test (comparison header carries no score bullet).
   Reverted for comparisons with a documented reason. Lesson: check where
   the data is displayed before deduplicating.
6. **Two nonsense `sed -i ... /dev/null` commands** — harmless no-ops from
   habit; embarrassing nonetheless.
7. **Todo granularity too coarse** (M10+M11 merged, M2+M3 merged): F21,
   F33, F42 fell out of tracking and were only caught by writing this
   report.
8. **Prompt-legend fix was wishful**: attempt #1 assumed the model was
   confused by notation; the regeneration disproved it. The unwrap repair
   is the real fix (and is the currently-unverified stop point).

## e) WHAT WE SHOULD IMPROVE

- **Probe-verify every guard** (the `--untracked` catch is the proof of
  value).
- **Capability-check third-party libs with a 30-second scratch repro**
  before building on them.
- **Real-model feedback loop earlier** for anything prompt-adjacent — the
  `*` quirk would have surfaced in minute one instead of after the
  builders/prompt work.
- **Track F-task granularity in todos**, not M-task — three sub-items
  fell through.
- **Don't chain writes behind builds**; run build, then edit.
- Record benchmark/coverage numbers in a doc the moment they're measured
  (they rot otherwise).
- Activation configs belong somewhere durable (`/etc` or the repo's
  `docs/`), not `/tmp`.

## f) NEXT (up to 50, priority order)

1. Finish the `*`-unwrap: `go build ./... && go test ./pkg/vision/a2ui/`,
   fix fallout, re-run `-a2ui` against llama, validate output with the
   official-schema validator (target: ALL LINES VALID).
2. Full verification matrix: build/vet/gofmt/`test -race ./...`,
   jsonv2 regime, `go mod verify`, `go mod tidy -diff`.
3. `golangci-lint run ./...` (whole repo, not per-package).
4. Nix half: `nix run .#test`, `nix run .#lint`, `nix build .`,
   `nix build .#visionreviewd`, `nix flake check`.
5. M21: `docs/A2UI.md` — wire vs inference format table, lifecycle
   diagram, error taxonomy, model-quirk notes (incl. the `*` unwrap).
6. M21: `pkg/vision/a2ui/README.md` for pkg.go.dev browsers.
7. F33: builder-coverage table in FEATURES.md (all 18 kinds now covered).
8. F42: README `-a2ui` usage snippet.
9. F21: record current a2ui coverage + benchmark numbers in AGENTS.md.
10. M23 F63: `journalctl -k` around the 08-18 OOM window; classify.
11. M23 F64: `docs/BUILDFLOW.md` (OOM retry policy + json/v2 guard
    explainer).
12. M27 F71: extract `vendorHash` to `vendorHash.nix`; `nix build` verify.
13. M27 F72: lint-noise policy table (MD013, codespell, go-structure
    assets, go-licenses).
14. M26 F70: CI job building/testing WITHOUT `GOEXPERIMENT=jsonv2`.
15. M26 F69: reset `version` to `0.7.0-dev` in both binaries (after M24
    decision).
16. M25 F67: probe nixpkgs for go 1.26.6.
17. M25 F68: bump go.mod toolchain + flake if shipped; full matrix.
18. Decide `Decompile` wire-shape docs placement (A2UI.md section).
19. Docs harvest: remove completed items from TODO_LIST → CHANGELOG
    `[Unreleased]` (Added: conformance suite, builders, Decompile,
    Theme/DataModel, `-a2ui`, fuzz, benchmarks; Changed: strict envelope,
    null-vs-omit semantics, exhaustruct scope; Fixed: Tabs missing,
    score duplication, llama readiness race).
20. Annotate the Pareto plan file with completion hashes (never rewrite).
21. Commit in reviewable slices (conformance+guard / builders+decompile /
    daemon fixes+replay BDD / docs+glossary).
22. Push + watch CI green (the grep guard's first real run).
23. Move activation configs out of `/tmp` into a durable location.
24. Consider a `visionreviewd compare` real run (touch one golden →
    auto-compare fires) — 0 compared so far this session.
25. `visionreviewd events` sanity over the real store; `SummarizeEvents`.
26. Wire the full 216-view DiscordSync glob into a durable config; decide
    interval; estimate pass duration (~30 s/view on CPU → ~2 h).
27. SystemNix activation instructions (for the user to run with sudo).
28. Add llama `--image-min-tokens 1024` consideration (upstream Qwen-VL
    grounding warning seen in llama-server logs).
29. Real-model a2ui on 2–3 more screenshots (mobile viewport, dark theme)
    to exercise Image/Icon/variants.
30. Check `reports/jscpd-report.json` / `.art-dupl-baseline.json` freshness
    after the scan (possible stale artifacts).
31. STATUS: this report's open questions resolved → fold answers into
    TODO/ROADMAP.
32. Consider `Retract`/tag hygiene follow-ups per M24 answer.
33. pkg.go.dev: confirm santhosh-tekuri addition doesn't disturb the
    module graph for consumers (`go mod graph` eyeball).
34. (Optional) conformance test cache: compile official schemas once via
    `sync.Once` if test runtime ever matters (~60 ms now — skip unless
    slow).
35. (Optional) ginkgo `DescribeTable` for the builder wire-shape table.

## g) QUESTIONS (cannot resolve myself)

1. **Release presentation (M24, gates M26):** delete ghost tags
   `v0.2.1`/`v0.3.0` on origin, and promote v0.6.1 out of prerelease / cut
   a synced v0.6.2 with this session's work — which combination do you
   want? (Destructive + product call.)
2. **Activation depth:** prepare SystemNix host-enablement steps for you
   to run with sudo (config under `/etc`, module on, doctor gate), or is
   the user-space llama + `once`/daemon setup sufficient for now?
3. **The `*`-unwrap policy:** keep it as a Generate-time normalization
   (current design: repair, then validate), or reject star-nested output
   as a model error instead? And is nsfwcaption-qwen3-vl the long-term
   model choice — i.e. how much should we invest in its quirks?

---

_Point-in-time report. Next session: resume at b.1 (unwrap verification),
then f.2–f.4 verification matrix, then the harvest/commit chain._
