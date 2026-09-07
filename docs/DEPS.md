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

Snapshots shorten stream replay. At current scale replay of a full
journal is milliseconds (see M14 baseline, `docs/status/` follow-up) and
streams are short. Revisit if the perf baseline shows replay cost
becoming user-visible (rough trigger: >100 ms full-journal replay or
streams routinely exceeding ~10k events).

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
