GO ?= go
BIN_DIR ?= bin
BIN := $(BIN_DIR)/linstor-mcp-server
PKGS := ./...
MAIN := ./cmd/linstor-mcp-server

.PHONY: all build test test-race coverage fuzz-smoke fmt vet verify check check-full clean run

all: check

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN) $(MAIN)

test:
	$(GO) test $(PKGS)

test-race:
	$(GO) test -race $(PKGS)

coverage:
	$(GO) test -coverprofile=coverage.out -covermode=atomic $(PKGS)
	$(GO) tool cover -func=coverage.out

fuzz-smoke:
	$(GO) test ./internal/app -run='^$$' -fuzz=FuzzDecodeCursor -fuzztime=3s
	$(GO) test ./internal/app -run='^$$' -fuzz=FuzzParseResourceURI -fuzztime=3s

fmt:
	$(GO) fmt $(PKGS)

vet:
	$(GO) vet $(PKGS)

verify:
	$(GO) mod verify

check: fmt vet verify test build

check-full: check test-race coverage fuzz-smoke

clean:
	rm -rf $(BIN_DIR) coverage.out

run: build
	./$(BIN)
