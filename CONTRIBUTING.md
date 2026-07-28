# Contributing

Contributions are welcome. This document covers the conventions and tooling
you need to know before opening a PR.

## Prerequisites

- **Go 1.26+** (see `go.mod` for the exact version)
- **Nix flakes** (optional but recommended for reproducible builds)

## Build & Test

```bash
go build ./...              # Build everything
go test -race ./...         # Run all tests with race detector
go vet ./...                # Run go vet
golangci-lint run ./...     # Lint (130+ linters)
gofumpt -l .                # Check formatting (must be empty)
go mod tidy                 # Verify go.mod/go.sum are tidy

# Nix flake (reproducible)
nix run .#test              # go test -race -v -coverprofile=coverage.out ./...
nix run .#lint              # golangci-lint run ./...
nix build .                 # Build the package
nix flake check             # Canonical quality gate
```

All of the above must pass before merging. CI runs them automatically.

## Code Conventions

### Error Handling

- **Sentinel errors** live in `pkg/errors/` and are re-exported from `pkg/vision/`
  for backwards compatibility
- **Dynamic errors** must wrap static sentinels: `fmt.Errorf("%w: ...", errFoo)`
  (enforced by `err113`)
- **Model errors** are classified into `*ModelError` with an `ErrorKind` and
  `IsRetryable()` for consumer retry logic
- **Error wrapping**: internal helpers return raw errors; callers classify.
  See `wrapcheck` config for exceptions

### Testing

- **Table-driven tests** for pure functions (config validation, image format detection)
- **BDD tests** (Ginkgo + Gomega) for user-facing behavior
- **Testify tests** for error classification and feature tests
- **Fuzz tests** for input processing (`FuzzDetectImageFormat`, `FuzzResizeImage`, etc.)
- **All test functions must call `t.Parallel()`** unless they use `t.Setenv()`
  or mutate global state (enforced by `paralleltest`)
- **Float comparisons**: use `require.InDelta` or `require.InEpsilon`, never
  `require.Equal` on floats (enforced by `testifylint`)
- **Mock model**: tests use a shared `mockModel` with error injection + call counting

### Style

- **No magic numbers**: extract named constants (enforced by `mnd`)
- **Descriptive variable names**: minimum length enforced by `varnamelen`
  (common short names like `err`, `ok`, `i`, `x`, `y` are ignored)
- **Exhaustive struct literals**: all fields must be set explicitly (enforced by
  `exhaustruct`, with documented exceptions for external types)
- **Early returns** over nested conditionals
- **Composition over inheritance**

### Linting

The project uses **golangci-lint v2** with ~130 linters. Key rules:

- **No `pkg/vision/` exclusions** — the core SDK is fully linted
- **Path-based exclusions** are documented with rationale in `.golangci.yaml`
- **`//nolint` directives** must include a comment explaining WHY
- Run `golangci-lint config verify` to validate the config

## Pull Request Checklist

- [ ] All tests pass: `go test -race ./...`
- [ ] Lint is clean: `golangci-lint run ./...`
- [ ] Formatting is clean: `gofumpt -l .` (empty output)
- [ ] `go mod tidy` produces no diff
- [ ] New code is documented (godoc comments on exported symbols)
- [ ] New features have tests
- [ ] `CHANGELOG.md` updated under `[Unreleased]`
- [ ] `FEATURES.md` updated if feature status changed
