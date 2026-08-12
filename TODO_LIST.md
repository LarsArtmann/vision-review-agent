# TODO List

Short- and mid-term actionable tasks. Each item is bounded and scoped.
For long-term ideas and raw direction, see [ROADMAP.md](ROADMAP.md).
For current feature inventory, see [FEATURES.md](FEATURES.md).

> **Discipline:** When a task is completed, **remove it from this file** and
> record it under `[Unreleased]` in [CHANGELOG.md](CHANGELOG.md). This file is
> for open work only — no `[x]` checkboxes, no "Previously Completed" sections.

---

## Release mechanics

- [ ] **Resolve tag anomaly** — `v0.2.1` points to commit `d5dda4b` (a
      pre-v0.2.0 test-formatting commit). The `v0.3.0` ghost on the same commit
      was already deleted during the `v0.4.0` release. `v0.2.1` remains and
      should be deleted for a clean history.
      **Destructive — requires explicit user approval (force-push / tag deletion).**

## Open questions (from ROADMAP — need product decisions before becoming tasks)

- **Structured hooks payload: is a breaking API change acceptable?** The
  nil-`RawResponse` hazard in structured `fireFinish` is now documented (see
  `AnalyzeResult.RawResponse`), but a proper fix (discriminated `HooksEvent`
  struct or generic `StructuredHooks[T]`) would be a breaking `Hooks` change.
- **Semver policy for 0.x** — is 0.x "anything goes" or semver-lite? Should
  breaking changes get a `### Breaking` callout in CHANGELOG?
