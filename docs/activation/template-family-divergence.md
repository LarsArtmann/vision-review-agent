# Template-Family Divergence Report — 14 Astro Sites

- **Date:** 2026-09-19
- **Scope:** art-dupl, clean-wizard, cmdguard, dynamic-markdown-site,
  go-atomic-write, go-branded-id, go-error-family, go-filewatcher,
  go-output, go-workflow-auditlog, gogenfilter, md-go-validator,
  samber-do-auditlog, typespec-asyncapi (the Astro/Tailwind landing-page
  family; templ-components, emeet-pixyd and learnings use other stacks)
- **Method:** pairwise `difflib.SequenceMatcher` ratios over
  `layouts/LandingLayout.astro`, `components/HeroSection.astro`,
  `styles/global.css`; token-name intersection on the CSS.

## Numbers

| File                  | Lines (range) | Mean pairwise ratio | Max pair                                                          |
| --------------------- | ------------- | ------------------- | ----------------------------------------------------------------- |
| `LandingLayout.astro` | 66–82         | **0.875**           | 0.955 (go-filewatcher ~ go-workflow-auditlog)                     |
| `HeroSection.astro`   | 72–132        | 0.546               | 0.829 (go-error-family ~ go-workflow-auditlog)                    |
| `global.css`          | 97–168        | 0.613               | **1.000** (go-filewatcher ~ go-workflow-auditlog, byte-identical) |

Core design-token intersection: only **5 / 14** repos define all of
`--color-bg-primary`, `--color-text-primary`, `--color-accent`,
`--color-accent-hover`, `--color-on-accent`, `--color-border`,
`--color-text-muted`. Nine repos lack `--color-on-accent` — the exact
token whose absence caused the 2026-09-19 contrast violations.

## Interpretation

1. **The layout skeleton is effectively one file copied 14 times** (87.5 %
   mean similarity, no repo byte-identical — every copy has drifted).
2. **Hero sections genuinely diverge** (0.546): per-site copy, badges,
   install commands, demo embeds. This divergence is _by design_ — the
   hero is the per-product surface.
3. **The token layer is where drift hurts**: today's fixes were
   ×14 edits for shared classes (empty `th` ×9, scrollable-region
   tabindex ×7, `--color-on-accent` token ×3 of 14 needed).

## Failure history (why this matters)

Every family-wide quality action in 2026-09 cost a 14-repo sweep with
silent-coverage risks (typespec's Go `src/` shadowed `website/src` in run
1): a11y fixes, og:image injection, pnpm v11 allowBuilds, poster webp.
The M22 plan question "extract package vs sync script" exists because
this cost recurs monthly.

See [the ADR](../adr/0001-template-family-sync-over-extract.md) for the
decision.
