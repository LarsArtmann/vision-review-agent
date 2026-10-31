# Dependencies & Journal Wire Contract

How this repo consumes `go-cqrs-lite`, what the on-disk event journal
contract is, and how to bump the dependency without endangering existing
journals. The golden-journal suite (`internal/reviewd/golden_journal_test.go`)
machine-pins everything described here; when this doc and a test disagree,
the test wins and this doc is stale.

Sources verified against `go-cqrs-lite` (event/v4 v4.9.0,
storage/bbolt v4 v4.1.0, decider/v4 v4.5.0) and `go-codec` v0.2.0 module
sources, 2026-09-07.

---

## The journal wire contract

The visionreviewd journal is a bbolt database (`<dataDir>/events.db`,
bucket pair `events` + `journal`). Every row is an upstream
`serializableEvent` CBOR envelope with EXACTLY this key set (sorted,
JSON-tag names):

```text
aggregate_id  aggregate_type  encoding  id  metadata
occurred_at   payload         schema_version  type  version
```

Pinned facts (each enforced by a test in `golden_journal_test.go`):

- `schema_version` is `1` on every row written by storage/bbolt
  v4.0.0 and v4.1.0 (byte-identical envelopes across both).
- `encoding` is `"cbor"` — payloads are CBOR by default
  (`event.DefaultCodec = codec.CBORCodec{}` in event/v4), and the first
  payload byte lands in the CBOR map range `0xa0–0xbf`.
- The envelope is CBOR, with a JSON-fallback read path in the upstream
  read path keyed off that first byte.

## The codec split: `codec/v4` → `go-codec`

Codecs no longer live inside go-cqrs-lite. `event/v4` imports the
separate module **`github.com/larsartmann/go-codec`** (currently v0.2.0)
which provides `CBORCodec`, `JSONCodec`, encoding autodetection, and the
envelope helpers. Consequences for this repo:

- `go-codec` appears in our `go.mod` as a DIRECT requirement (pin tests
  import `codec.EncodingCBOR`), not merely transitive.
- Events are self-describing: each row stamps its encoding, and
  `event.DecodePayloadAuto[T]` picks the codec per event. A journal may
  legitimately contain mixed JSON/CBOR rows; the daemon never assumes
  which.

## CBOR pitfall: `time.Time` identity

CBOR round-trips a `time.Time` to a DIFFERENT `time.Location` of the
same instant. Never compare deserialized timestamps with `==`; use
`.Equal`. This bit the golden tests during M2 and is pinned there
(`golden_journal_test.go` uses `.Equal` throughout).

## schema_version and evolution

`schema_version = 1` is what every upstream writer stamps today. If a
future bump changes the envelope (new key, different stamp), the
envelope-contract test fails — that failure is the designed early
warning, not a nuisance:

1. Read the upstream change; decide whether old journals must remain
   readable (they usually must — the journal is the source of truth).
2. If the change is intentional and readable-both-ways, regenerate the
   golden fixture with the new library version:

   ```bash
   REVIEWD_GENERATE_GOLDEN_JOURNAL=1 go test ./internal/reviewd -run TestGenerateGoldenJournal
   ```

   and update `testdata/golden-journal.PROVENANCE.md` (versions, date,
   reason).
3. If old journals would NOT read under the new version, the bump is
   blocked: record the incompatibility and wait for (or contribute) a
   migration path upstream.

## Bump procedure (condensed)

Full matrix: AGENTS.md "Build & Test Commands". The dependency-specific
gates:

1. Baseline before touching anything: `nix run .#verify-bump`.
2. Bump, then `nix run .#verify-bump` (build, vet, fmt, lint,
   cache-free race tests, `go mod tidy -diff`, `go mod verify`).
3. `nix run .#update-vendor-hash` — repairs `vendorHash.nix` and proves
   the rebuild (no-ops when nothing moved).
4. The golden-journal suite + `TestNoV5RemovedPairFormAPIs` guard are
   part of `go test ./...`; they are the wire-contract and v5-readiness
   gates. Do not skip or `//nolint` them.
5. Record the re-audit result (wire diff) in the bump's CHANGELOG entry.

## ADR: upstream capabilities — adopt or decline (2026-09-07)

Evaluated against the daemon's actual workload: ONE bbolt handle, ONE
writer goroutine (the pass loop), journal-as-source-of-truth, markdown
as a rebuildable projection.

### `WithBatchCommit` (storage/bbolt v4.1.0) — DECLINED

Routes writes through bbolt `db.Batch` (group commit). Upstream's own
contract: "only worthwhile with concurrent writers — a lone caller gains
nothing." The daemon is definitionally single-writer, and the single
bbolt handle serializes writers anyway. Measured (single goroutine,
2000 256-byte puts, local disk):

| mode        | per-op    | total   |
| ----------- | --------- | ------- |
| `db.Update` | 9.9 µs    | 19.8 ms |
| `db.Batch`  | 10 241 µs | 20.5 s  |

`db.Batch` pays a ~1000× penalty with no batch-mates to group with.
Adopting this would have been a silent 1000× write-path regression.
Revisit only if the daemon ever grows concurrent writers.

### `DecorateJournal` / journal middleware (event/v4 v4.9.0) — DECLINED

Read-side transform decoration (per-chunk of 128 events on streaming
reads). Clean API, but the daemon reads via `Store.AllEvents` and folds
explicitly (`VerifyJournalEvents`, `Replay`); it has no
filtering/redaction/mapping need on the read path. Available if one
appears.

### query/v4 read models — DECLINED

The bbolt `Backend` exposes KV read models; the daemon's INDEX could be
served from one. But the daemon's design keeps bbolt journal-only and
treats markdown (+INDEX) as the projection, rebuilt deterministically by
`Replay`. A KV read model would duplicate the ViewState fold and create
a second state to keep consistent. The journal + fold IS the read model.

### snapshot/v4 — DECLINED (revisit trigger recorded)

Snapshots shorten stream replay. Measured baseline (2026-09-07,
`internal/reviewd/perf_test.go`, numbers in
`docs/status/2026-09-07_22-15_perf-baseline-m14.md`): full-journal
replay of 10k events costs ~200 ms and stays off the hot path — replay
runs on demand (`visionreviewd replay`, doctor), never per pass or per
tick. A snapshot would add a second state to keep consistent for a
~200 ms one-off saving. Revisit only when journal scale approaches
~1M events (extrapolating linearly: ~20 s replay) or replay moves onto
the per-pass hot path.

### metadata/v4 — DECLINED

Per-event metadata columns; every field the daemon needs already lives
in the typed payloads. Nothing to put in metadata.

## Known upstream defects (observed, 2026-09-07)

- `OpenWith(path, &bolt.Options{ReadOnly: true}, …)` is unusable:
  bucket creation is unconditional, so read-only opens ALWAYS error.
- Without a `Timeout`, an open against a held journal blocks forever
  (no flock deadline). This is why `BackupJournalFile` and
  `JournalLockHeld` open raw bbolt with bounded timeouts instead of
  going through upstream's Store/Backend.
- Candidate upstream asks: contract-test coverage for the
  `serializableEvent` envelope (our golden method is the repro),
  per-module CHANGELOGs, and the read-only defect above.

## json/v2 dual-regime blocker: upstream watch (2026-10-31)

`kaptinlin/jsonschema` v0.9.9+ imports `encoding/json/v2` in core files and
`charm.land/fantasy` ≥ v0.45.2 floors it, which makes the
`GOEXPERIMENT=none` v1-regime build impossible for every package that
reaches fantasy (verified 2026-10-31: that is ALL of them — the
fantasy-free package set is empty). Watch points, cheapest first:

| Option | Trigger | Cost |
|---|---|---|
| kaptinlin ships a dual-regime release (v2 imports moved behind build tags) | upstream release notes / go-cqrs-lite-style regime split | zero — `GOEXPERIMENT=none` build starts passing; restore the full no-jsonv2 SDK scope (the CI watchdog goes red to remind us) |
| fantasy pins kaptinlin < v0.9.9 | fantasy CHANGELOG | zero — same restoration |
| Pin fantasy v0.41.1 (last kaptinlin-v0.9.8) | we decide the v1 regime matters more than fantasy updates | permanent drift against the upgrade daemon; REJECT unless the ecosystem stalls for months |

The 2026-10-31 CI conversion (no-jsonv2 → json-v1 watchdog) makes option
1/2 self-announcing: the job fails with "blocker is GONE" the moment the
transitive chain clears, so no human calendar is needed.

## Seventh-recurrence post-mortem (2026-10-11 → 2026-10-31)

`fad59ce` (auto-commit "Daemon: true") re-migrated 7 files to
v2/jsontext; it sat undetected for three weeks because detection was
memory-based (AGENTS.md knowledge) rather than scheduled, and one file
(`fakeserver_test.go`) had in fact been dirty since `6390b14` (09-29) —
the 2026-10-05 "grep clean" claim had missed it. Full narrative:
AGENTS.md dual-json bullet, SEVENTH recurrence paragraph. Guards landed
2026-10-31: package + module-wide guard tests (bypass-immune), weekly
scheduled grep workflow, watchdog CI job, `.go-auto-upgrade.json`
migrator exclusion.

## Quantization tradeoff: Q8_0 vs smaller (open decision, 2026-10-31)

The default model ships as Q8_0 (~9-10 GB download). Smaller quants
(Q4_K_M et al) would shrink the llama-server first-start download and the
restart/recovery window after a host reboot, at a fidelity cost that has
NEVER been measured for this review prompt (score stability, markdown
structure). Decision left open: measure score variance + review quality
on one fleet corpus across quants before switching the default. Until
then Q8_0 stays: fidelity is the product; recovery speed is tunable via the
module's `watchdogSec` option (nixos/visionreviewd.nix) instead.
