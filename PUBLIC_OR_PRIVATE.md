# Public vs. Private: Vision Review Agent

**Date:** 2026-05-04 | **Verdict: Make it PUBLIC (conditional)**

---

## Executive Summary

Vision Review Agent is a well-structured, clean Go SDK that wraps `charm.land/fantasy` to provide a focused vision/image analysis API. It is small (2,130 LOC), has no secrets, no proprietary algorithms, no competitive moat to protect, and fills a genuine gap in the Go ecosystem. The arguments for going public significantly outweigh the reasons to stay private.

---

## Project Assessment

| Dimension | Rating | Notes |
|---|---|---|
| Code Quality | **B+** | Clean, idiomatic Go. Good error handling, table-driven tests, proper pkg/internal split. Minor lint warnings (exhaustruct, paralleltest) but nothing critical. |
| Documentation | **A** | Excellent README with quick start, CLI usage, SDK examples, config table, error types. AGENTS.md with architecture decisions. CHANGELOG, AUTHORS. |
| Test Coverage | **A-** | Comprehensive table-driven tests with mocks. Good edge case coverage. Coverage threshold enforced at 70%. |
| Architecture | **A-** | Clean separation: `pkg/vision` (public SDK), `pkg/errors` (centralized errors), `internal/visionutil` (private helpers), `cmd/vision` (CLI). Follows Go conventions. |
| Security Posture | **A** | No secrets, no hardcoded keys, no sensitive data. API keys from env vars only. Magic byte validation for images. |
| Ecosystem Fit | **B+** | Fills a real need: vision SDK for Go built on Charm's ecosystem. Few comparable libraries exist. |
| Maturity | **C+** | v0.1.0, single contributor, no consumers yet. CHANGELOG is sparse. Package build in flake.nix is commented out. |

---

## PRO: Arguments for Making Public

### 1. No Competitive Moat to Protect
The code is a thin, well-designed wrapper around `charm.land/fantasy`. There are no proprietary algorithms, no trade secrets, no unique IP. The value is in the **convenience and curation**, not in hidden knowledge. Keeping it private protects nothing.

### 2. Community Value in the Go + AI Ecosystem
Go is underrepresented in AI/vision tooling compared to Python and TypeScript. A clean, idiomatic Go SDK for vision analysis — with streaming, structured output, multi-provider support, and a CLI — is genuinely useful. The Charm ecosystem (Bubbletea, Lip Gloss, Fantasy) has an enthusiastic community that would benefit from and appreciate this.

### 3. Portfolio & Reputation
For the author, this is a strong portfolio piece: clean architecture, proper testing, good documentation, Nix flake, linting. Making it public demonstrates engineering quality to potential employers, clients, or collaborators.

### 4. Adoption & Feedback Loop
Open-sourcing enables:
- Bug reports from real-world usage
- Feature requests driven by actual use cases
- Potential contributors improving the SDK
- Organic growth through Go module discovery (`pkg.go.dev`)

### 5. Dependencies Are Already Public
The core dependency (`charm.land/fantasy`) is public. The patterns used (table-driven tests, builder pattern, error re-exports) are standard Go. There's nothing in here that isn't already derivable from reading fantasy's docs.

### 6. No Security Risk
- Zero secrets in the codebase
- API keys only via environment variables
- No internal infrastructure details exposed
- No customer data or PII

### 7. Aligns with Charm Ecosystem Culture
The Charmbracelet ecosystem thrives on open-source contributions. This project naturally extends that ecosystem. Keeping it private is culturally misaligned.

---

## CONTRA: Arguments for Staying Private

### 1. Immature — v0.1.0 with No Real Users
Publishing an SDK nobody uses yet could create a poor first impression. The API surface might need breaking changes once real usage patterns emerge. There's risk of publishing "abandonware" if momentum doesn't build.

### 2. Current License Contradiction
The `LICENSE` file says **PROPRIETARY**, but the README says **MIT**, and `flake.nix` references `licenses.mit`. This must be resolved before going public. Publishing under a proprietary license would defeat the purpose.

### 3. Maintenance Burden
Going public creates implicit obligations:
- Responding to issues and PRs
- Maintaining backward compatibility
- Keeping dependencies updated publicly
- Documentation upkeep

If the author isn't ready to commit to this, a public repo with slow response times is worse than a private one.

### 4. Flake.nix Build is Incomplete
The package build in `flake.nix` is commented out (`# packages.default = vision-review-agent`). This signals "not ready for public consumption" in the Nix ecosystem.

### 5. Minor Polish Gaps
- `.gitignore` has formatting issues (line 31-36 lack `#` prefixes)
- `ScreenshotAnalyzer` creates a new `Agent` on every call (no reuse)
- `fullText` concatenation in `AnalyzeStream` is O(n²) — should use `strings.Builder`
- Deprecated `VisionAgent` alias still exists
- No `go vet` or race detector in CI
- No GitHub Actions / CI pipeline visible

### 6. Naming & Branding Risk
The name "vision-review-agent" suggests an agent framework rather than an SDK. If the project pivots or grows, the name may not age well. A public rename is more painful than a private one.

---

## Conditional Verdict: Make It Public

**Go public, but address these conditions first:**

### Must-Do Before Publishing (blocking)

| # | Action | Effort |
|---|---|---|
| 1 | **Fix LICENSE** — Replace proprietary license with MIT (matching README and flake.nix) | 5 min |
| 2 | **Remove AGENTS.md** — Contains internal workflow instructions, not user-facing docs. Move to `.github/` or keep local-only | 10 min |
| 3 | **Remove docs/status/** — Internal status reports shouldn't be public | 2 min |
| 4 | **Review git history** — Ensure no secrets were ever committed (currently clean, but verify) | 5 min |
| 5 | **Fix .gitignore** formatting | 2 min |

### Should-Do Before Publishing (recommended)

| # | Action | Effort |
|---|---|---|
| 6 | Fix `fullText` O(n²) concatenation → `strings.Builder` | 10 min |
| 7 | Uncomment and complete flake.nix package build | 30 min |
| 8 | Add `CONTRIBUTING.md` with guidelines | 30 min |
| 9 | Add GitHub Actions CI (test, lint, build) | 1 hour |
| 10 | Remove deprecated `VisionAgent` alias or properly document deprecation path | 5 min |

### Nice-to-Have After Publishing

| # | Action | Effort |
|---|---|---|
| 11 | Add Go module badge, CI badge to README | 15 min |
| 12 | Publish to `pkg.go.dev` by tagging v0.1.0 | 5 min |
| 13 | Add `Agent` pooling/caching in `ScreenshotAnalyzer` | 1 hour |
| 14 | Add more provider examples (Anthropic direct, Google Gemini) | 2 hours |
| 15 | Consider renaming to something more SDK-like (e.g., `go-vision`, `visionsdk`) | — |

---

## Risk Matrix

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Low adoption / obscurity | Medium | Low | Market in Go + Charm communities; tag releases |
| Breaking API changes needed | High | Medium | Stay at v0.x; use Go module conventions |
| Maintenance burden grows | Low | Medium | Set expectations in README ("best effort") |
| Someone forks and competes | Very Low | Very Low | Not a competitive product; forks are fine |
| Security issue in dependency | Low | Medium | Enable Dependabot / Renovate |

---

## Conclusion

This project has **no reason to stay private**. It contains nothing proprietary, nothing sensitive, and nothing that benefits from obscurity. The code quality is good enough to represent the author well. The Go + AI vision space needs more quality tooling. The only real blocker is the license contradiction — fix that, clean up internal docs, and publish.

**Recommendation: PUBLIC — after completing the 5 must-do items above.**
