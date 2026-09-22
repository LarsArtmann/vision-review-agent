# ADR 0001: Template family — sync script over extract package

- **Date:** 2026-09-19
- **Status:** Accepted
- **Context:** [template-family-divergence.md](../activation/template-family-divergence.md)

## Decision

Keep the 14 landing-page repos as independent copies and standardize them
with a **sync/check script** living in `vision-review-agent`
(`scripts/family-a11y-guard.sh` is the first slice). Do NOT extract a
shared Astro component package.

## Why not extract-package

1. **Version skew would replace drift as the failure mode.** 14 repos on
   independent release/deploy cycles would each pin their own package
   version; a fix lands only after 14 bumps and deploys — strictly more
   work than today's sweep, plus a new dependency to maintain.
2. **The divergent part is the part that must stay per-repo.** Hero
   sections measure 0.546 mean similarity: copy, badges, install
   commands, demo embeds are product surface, not shared UI. A package
   would either be too generic to help or too specific to fit.
3. **The identical part is tiny.** The shared skeleton is ~70 lines of
   layout. A one-file npm dependency (or git subtree) for 70 lines adds
   machinery disproportionate to the deduped volume.
4. **Precedent:** templ-components (the Go-side shared-component repo)
   was adopted for templ/HTMX UI, but the landing sites were deliberately
   kept framework-static. Adding a JS package chain to 14 static sites
   works against that shape.

## Why sync-script

1. **It matches how the family already evolves** (copy-mutate), but makes
   the copy step _audited_: a script applies a canonical change and
   reports coverage against the repo list, so the
   "found-but-unfixed = exit 1" rule is mechanical, not attentional.
2. **No new dependency graph.** Repos stay buildable standalone.
3. **Check-mode is cheap CI.** `--check` (report only) can run in any
   repo's pipeline or across the fleet without touching sources.

## First slice (prototype, F85)

`scripts/family-a11y-guard.sh --check`: per family repo, flags
(a) `<th>` with no text content in the comparison-table component,
(b) scrollable `pre`/`code`/`.overflow-*` elements without `tabindex="0"`,
(c) accent CTA without an `--color-on-accent`-backed class.
Exit 1 if any repo has findings — the coverage-diff guard from the
2026-09-19 a11y sweep, made repeatable.

## Consequences

- Family-wide changes remain sweeps, but scripted ones with coverage
  output.
- New repos cloned from the family inherit the guard as a post-clone
  check.
- Revisit extract-package only if a _behavioral_ component (not copy)
  starts changing weekly across all repos.
