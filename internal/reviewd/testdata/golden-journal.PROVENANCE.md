# Golden Journal Fixture Provenance

`golden-journal.bbolt` is a frozen bbolt event store produced by a real
`reviewed.Store`. It is the regression contract for journal readability:
every dependency bump must keep these exact bytes loading and folding
identically. `golden_journal_test.go` enforces that.

## Contents

Two view streams, 7 events total:

1. `discordsync:Settings--dark--desktop` — `view.captured`, `view.reviewed`,
   `view.captured`, `view.compared`, `view.reviewed` (versions 1–5; covers
   capture invalidation, score trend 6 → 8, one comparison)
2. `visionreview:Index--light--mobile` — `view.captured`, `view.reviewed`
   (versions 1–2)

## Fingerprint

- Size: 65536 bytes
- SHA-256: `7dcca98808b53b244428c99ad3e61b3e40d192151992ab33508b61132097e194`

Tests assert the bytes never change; the fingerprint here is for humans.

## Created

- 2026-09-07, commit `556065b` (post go-cqrs-lite full bump)
- Go `go1.26.7`, `GOEXPERIMENT=jsonv2`
- Producer dependency set (from `go.mod`):
  - `go-cqrs-lite/event/v4` v4.9.0
  - `go-cqrs-lite/decider/v4` v4.5.0
  - `go-cqrs-lite/storage/bbolt/v4` v4.1.0
  - `go-cqrs-lite/id/v4` v4.5.0
- Wire format at creation: CBOR envelope of `serializableEvent`
  (`id`, `type`, `aggregate_id`, `aggregate_type`, `version`,
  `schema_version`=1, `payload`, `occurred_at`, `metadata`, `encoding`),
  payload encoding stamped `cbor` (event `DefaultCodec`), bucket
  `cqrs_events`.

## Regenerating

```bash
REVIEWD_GENERATE_GOLDEN_JOURNAL=1 go test ./internal/reviewd -run TestGenerateGoldenJournal -v
```

Only regenerate deliberately: the new bytes must still decode all four
assertions in `golden_journal_test.go` (fold, payloads, envelope contract,
payload JSON tags), and this file's fingerprint must be updated. If a bump
changes the wire format, the fixture stays as the OLD-format reader test and
a new fixture documents the new format — never overwrite the old one away.
