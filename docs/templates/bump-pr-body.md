# Dependency bump PR template

Copy this body for dependency-bump PRs. Fill the `<...>` placeholders; do
not merge with an unchecked verification box.

```markdown
## Summary

- Bump <dep> <from> → <to> (<why: CVE / feature / drift>)

## Verification

- [ ] `nix run .#verify-bump` (build/vet/gofmt/lint/race/tidy/verify)
- [ ] `nix build .#visionreviewd` (vendorHash in sync — if stale:
      `nix run .#update-vendor-hash`)
- [ ] GOEXPERIMENT=jsonv2 full suite + no-jsonv2 watchdog green
- [ ] govulncheck 0 reachable
```

Notes:

- `verify-bump` is the minimal gate; the full verification matrix lives in
  AGENTS.md ("Verification matrix") and is required for cross-cutting bumps
  (go-cqrs-lite, fantasy).
- Lint version policy: if the bump touches golangci-lint, CI and local must
  move in the SAME commit (see AGENTS.md "Lint version pin").
