BINARY := multiclaude
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X github.com/natikgadzhi/multiclaude/cmd.buildVersion=$(VERSION) \
	-X github.com/natikgadzhi/multiclaude/cmd.buildCommit=$(COMMIT) \
	-X github.com/natikgadzhi/multiclaude/cmd.buildDate=$(DATE)

.PHONY: build test lint vet ci install clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

test:
	go test -race ./...

lint:
	go vet ./...

vet:
	go vet ./...

ci: vet test

install:
	go install -ldflags "$(LDFLAGS)" .

clean:
	rm -f $(BINARY)
