default:
    @just --list

build:
    go build ./...

cli:
    go build -o vision-cli ./cmd/vision

test:
    go test ./pkg/... ./internal/... -v

test-race:
    go test ./pkg/... ./internal/... -race

coverage:
    #!/usr/bin/env bash
    set -euo pipefail
    threshold=70
    mkdir -p coverage
    go test ./pkg/... ./internal/... -coverprofile=coverage/coverage.out
    coverage=$(go tool cover -func=coverage/coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    echo "Coverage: ${coverage}%"
    # Compare using awk instead of bc
    if awk "BEGIN {exit !($coverage < $threshold)}"; then
        echo "ERROR: Coverage ${coverage}% is below threshold ${threshold}%"
        exit 1
    fi
    echo "Coverage check passed!"

vet:
    go vet ./...

fmt:
    gofmt -w .

clean:
    rm -f vision-cli coverage.out
    go clean ./...

lint:
    golangci-lint run ./...

structure-lint:
    go-structure-linter .

all: vet fmt test build
