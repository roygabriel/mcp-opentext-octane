VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
BINARY := mcp-opentext-octane

.PHONY: all build test cover vet lint clean run docker tools vuln

all: vet lint test build

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY) .

test:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo ""
	@go tool cover -func=coverage.out | grep ^total:

cover: test
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

vet:
	go vet ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY) coverage.out coverage.html

run: build
	./$(BINARY)

docker:
	docker build -t $(BINARY):$(VERSION) .

tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

vuln:
	govulncheck ./...
