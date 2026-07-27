# TODO List

Short- and mid-term actionable tasks. Each item is bounded and scoped.
For long-term ideas and raw direction, see [ROADMAP.md](ROADMAP.md).
For current feature inventory, see [FEATURES.md](FEATURES.md).

> **Discipline:** When a task is completed, **remove it from this file** and
> record it under `[Unreleased]` in [CHANGELOG.md](CHANGELOG.md). This file is
> for open work only — no `[x]` checkboxes, no "Previously Completed" sections.

---

## Release mechanics

- [ ] **Resolve tag anomaly** — `v0.2.1` and `v0.3.0` both point to commit
      `d5dda4b` (a pre-v0.2.0 test-formatting commit). Decide: delete and re-tag,
      or supersede with a real `v0.3.0` once `[Unreleased]` work is tagged.
      **Destructive — requires explicit user approval (force-push / tag deletion).**

## Open questions (from ROADMAP — need product decisions before becoming tasks)

- **catwalk or hand-rolled CLI providers?** The three providers added to
  `cmd/vision/main.go` (Anthropic, Google, openaicompat) work but will rot as
  fantasy evolves. Replace them with `github.com/charmbracelet/catwalk`, or
  keep hand-rolled and layer catwalk on top?
- **Structured hooks payload: is a breaking API change acceptable?** The
  nil-`RawResponse` hazard in structured `fireFinish` is now documented (see
  `AnalyzeResult.RawResponse`), but a proper fix (discriminated `HooksEvent`
  struct or generic `StructuredHooks[T]`) would be a breaking `Hooks` change.
- **Semver policy for 0.x** — is 0.x "anything goes" or semver-lite? Should
  breaking changes get a `### Breaking` callout in CHANGELOG?
