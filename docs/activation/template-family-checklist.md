# Template-Family Quality Checklist — mined from gogenfilter (8/8)

_Distilled 2026-09-19 from the gogenfilter website (the fleet's top-scoring
site in both review passes) plus the fleet-wide axe audit
(`2026-09-19_16-50_a11y-audit-17-sites.md`). Use as the review gate for any
new or edited site in the family._

## Design tokens (global.css)

- [ ] **Every theme defines `--color-on-accent`** with a DARK value
      (`#0a0908`-class), and accent-bg elements use `text-on-accent`, never
      `text-bg-primary` (white-on-mid-tone fails AA contrast — the fleet's
      most common axe finding).
- [ ] **Amber/accent text on the always-dark hero uses the BRIGHT palette
      value** (`#fbbf24`-class), not the light-theme body value
      (`#b45309` fails on dark).
- [ ] Light and dark palettes are both explicit; no `prefers-color-scheme`
      half-states.

## Semantics & a11y

- [ ] Comparison-table corner cell carries visible text (`Feature`), not an
      empty `<th class="tracking-wider">`.
- [ ] Every scrollable `<pre>` / code panel has `tabindex="0"` (axe
      scrollable-region-focusable).
- [ ] Header/banner landmark is NOT nested inside `<main>`; site owns its
      `<main id="main-content">` (see templ-components `Page`).
- [ ] Decorative SVGs `aria-hidden`; content SVGs get `role="img"` +
      `aria-label`.
- [ ] Heading levels increase by one (h1 → h2 → h3); feature cards use
      visual-size overrides, not skipped levels.

## SEO / sharing

- [ ] `og:image` (PNG/JPG 1200×630, never SVG) at `/og/home.png` +
      `twitter:card = summary_large_image`.
- [ ] `canonical` URL matches the actually-served domain (the learnings
      incident: canonical pointed at a domain that never resolved).
- [ ] `sitemap-index.xml` present (Astro) and robots.txt allows crawling.

## Capture hygiene (review pipeline)

- [ ] Screenshots shot with `lazyImageLoadingEnabled=false` (blank-card
      contamination class).
- [ ] **Dark-mode capture will actually show a dark page**: the site must
      respond to `prefers-color-scheme` (Astro family: explicit light+dark
      palettes; Docusaurus: `respectPrefersColorScheme: true` — that exact
      key; `respectColorScheme` does not exist and schema-fails) or be
      dark-by-design with a manual toggle only (cmdguard class — say so in
      the AGENTS.md, or the dark pass just re-shoots the light theme).
- [ ] og/social assets referenced with absolute URLs built from `siteUrl`.
- [ ] pnpm ≥ 11 sites: build-script approvals in `pnpm-workspace.yaml`
      (`allowBuilds: esbuild: true`), NOT `package.json pnpm.*` (ignored by
      pnpm v11; missing entry = `astro build` fails on missing esbuild).

## Known non-goals

- py-1/py-2 paddings on badges/copy buttons are fine; only primary CTAs need
  ≥44px touch height (all family heroes already use px-6 py-3 — verified
  2026-09-19; the model-claimed tap-target violations were false positives).
