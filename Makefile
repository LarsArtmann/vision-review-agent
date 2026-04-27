.PHONY: all build test test-race vet fmt clean cli examples

all: vet fmt test build

build:
	go build ./...

cli:
	go build -o vision-cli ./cmd/vision

test:
	go test ./... -v

test-race:
	go test ./... -race

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -f vision-cli
	go clean ./...

examples: cli
	@echo "Examples built. Run with:"
	@echo "  export OPENAI_API_KEY=your-key"
	@echo "  go run examples/openai/main.go screenshot.png"
