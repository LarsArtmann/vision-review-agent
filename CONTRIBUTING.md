# Contributing

Thanks for your interest in contributing!

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## Development Setup

This project uses [Nix flakes](https://nixos.wiki/wiki/Flakes) for reproducible
builds. Enter the development shell:

```bash
nix develop
```

If you don't use Nix, ensure Go 1.26+ and
[golangci-lint](https://golangci-lint.run/) are installed.

### Build and Test

```bash
# Go toolchain (preferred for fast iteration)
go build ./...
go test -race ./...
go vet ./...

# Nix flake (reproducible)
nix run .#test              # go test -race -v -coverprofile=coverage.out ./...
nix run .#lint              # golangci-lint run ./...
nix build .                 # Build the package
nix flake check             # Canonical quality gate
```

### Formatting

The project uses `gofumpt` and `goimports` (via treefmt in the flake). Check
formatting before committing:

```bash
gofmt -l .                  # Should output nothing
golangci-lint run ./...     # Must pass cleanly
```

### Code Style

- Follow existing patterns in `pkg/vision/`
- Table-driven tests for pure functions
- BDD (Ginkgo/Gomega) for user-facing behavior specs
- Document all exported types and functions
- Add `//nolint:linter // reason` only when a lint rule is genuinely wrong

## Reporting Issues

Please use GitHub Issues to report bugs or request features.
