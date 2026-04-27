default:
    @just --list

build:
    go build ./...

cli:
    go build -o vision-cli ./cmd/vision

test:
    go test ./... -v -coverprofile=coverage.out -coverpkg=./...

test-race:
    go test ./... -race

coverage:
    #!/usr/bin/env bash
    set -euo pipefail
    go test ./... -coverprofile=coverage.out -coverpkg=./...
    coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    echo "Coverage: ${coverage}%"
    if (( $(echo "$coverage < 70" | bc -l) )); then
        echo "ERROR: Coverage ${coverage}% is below threshold 70%"
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

all: vet fmt test build
