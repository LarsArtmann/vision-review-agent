# D1 — Fold Semantics: Tolerate-and-Ignore Unknown Event Types

**Date:** 2026-10-31
**Status:** Accepted and applied (store fold default; auto-commits `a283759` + `3beef61`)
**Decided by:** parallel session 2026-10-31, resolving the question opened in
`docs/status/2026-10-31_17-27_session-status-healthgate-jsonv2-restoration.md` (g1)

## Context

`ApplyViewState` folds `view.captured` / `view.reviewed` / `view.compared`
events into per-view state. Until 2026-10-31 its default case returned
`fmt.Errorf("fold: unknown event type: %s", ...)` while
`TestApplyViewStateIgnoresUnknownEventType` asserted unknown events are
IGNORED (forward compatibility for schema evolution). One of the two had to
be wrong; the contradiction was master's only red test and the repo's only
`err113` lint finding (the same line).

## Decision

**Tolerate-and-ignore.** The fold's default case skips unknown event kinds
and increments a new `ViewState.UnknownEvents` counter, so the skip is
observable instead of silent.

Rationale: journals are a lifetime asset. An old binary must be able to fold
a journal written by a NEWER binary (event kinds it does not know); failing
the fold would make every journal upgrade a replay outage for every
out-of-date binary that touches it. Strictness has no offsetting benefit:
unknown events carry no data the fold could corrupt — they are skipped, not
misinterpreted.

## Consequences

- Old journals stay replayable across schema evolution (the property the
  test always asserted).
- A rising `UnknownEvents` count after an upgrade is the signal that the
  running binary predates the journal's newest event kinds.
- The `err113` finding on the removed `fmt.Errorf` default case is gone;
  the pinned golangci-lint is at zero findings for this package.
- Replay/doctor callers that want to surface partial folds can read
  `ViewState.UnknownEvents` from the fold result.

## Revisit triggers

- If a future event kind must be REJECTED (not skipped) for safety — e.g. a
  destructive event type — reintroduce strictness per-kind via an explicit
  deny list, not a blanket default error.
- If projections other than ViewState need the same tolerance, reuse the
  counter pattern instead of inventing a second policy.
