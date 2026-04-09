GO         ?= go
BIN_DIR    ?= bin
BINARY     := $(BIN_DIR)/linstor-mcp-server
PKGS       := ./...
MAIN       := ./cmd/linstor-mcp-server
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE       := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

.DEFAULT_GOAL := help

.PHONY: all help build test test-race coverage fuzz-smoke lint fmt fmt-fix vet verify check check-full clean run

all: check

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

build: ## Build the binary with injected version metadata
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(MAIN)

test: ## Run package tests
	$(GO) test $(PKGS)

test-race: ## Run tests with the race detector
	$(GO) test -race $(PKGS)

coverage: ## Generate coverage output
	$(GO) test -coverprofile=coverage.out -covermode=atomic $(PKGS)
	$(GO) tool cover -func=coverage.out

fuzz-smoke: ## Run short fuzz smoke tests for parser-heavy paths
	$(GO) test ./internal/app -run='^$$' -fuzz=FuzzDecodeCursor -fuzztime=3s
	$(GO) test ./internal/app -run='^$$' -fuzz=FuzzParseResourceURI -fuzztime=3s

lint: ## Run golangci-lint
	@GOPATH=$$(go env GOPATH); \
	LINT=$$(command -v golangci-lint 2>/dev/null || echo "$$GOPATH/bin/golangci-lint"); \
	if [ ! -x "$$LINT" ]; then \
		echo "golangci-lint not found. Install v2.11.4:"; \
		echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v2.11.4"; \
		exit 1; \
	fi; \
	$$LINT run

fmt: ## Check formatting (fails if any files need formatting)
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Unformatted files (run 'make fmt-fix'):"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

fmt-fix: ## Auto-fix formatting with gofmt
	gofmt -w .

vet: ## Run go vet
	$(GO) vet $(PKGS)

verify: ## Verify downloaded modules
	$(GO) mod verify

check: fmt vet verify lint test build ## Run CI-parity checks

check-full: check test-race coverage fuzz-smoke ## Run extended validation

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) coverage.out

run: build ## Build and run the server
	./$(BINARY)
