#!/usr/bin/env bash
# Local dependency-bump gate: everything a bump must pass before it can be
# committed — build, vet, format, lint, cache-free race tests, and module
# hygiene. Mirrors the first six steps of the AGENTS.md verification matrix;
# the nix steps stay separate because they need the nix toolchain.
# Run from the repo root (the flake app `.#verify-bump` does).
set -euo pipefail

fail() {
	printf 'verify-bump: FAILED at %s\n' "$1" >&2
	exit 1
}

step() {
	printf '==> %s\n' "$1"
}

step 'go build ./...'
go build ./... || fail 'go build'

step 'go vet ./...'
go vet ./... || fail 'go vet'

step 'gofmt'
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
	printf '%s\n' "$unformatted" >&2
	fail 'gofmt -l reported unformatted files'
fi

step 'golangci-lint run ./...'
if command -v golangci-lint >/dev/null 2>&1; then
	golangci-lint run ./... || fail 'golangci-lint'
else
	printf 'verify-bump: golangci-lint not found, skipping (nix run .#lint covers it)\n' >&2
fi

step 'go test -race -count=1 ./...'
go test -race -count=1 ./... || fail 'race tests'

step 'go mod tidy -diff'
go mod tidy -diff || fail 'go mod tidy -diff (committed go.mod/go.sum are not tidy)'

step 'go mod verify'
go mod verify || fail 'go mod verify'

printf 'verify-bump: all gates green\n'
