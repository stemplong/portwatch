# Makefile for portwatch
# Provides common development and build tasks

BINARY_NAME := portwatch
BUILD_DIR   := ./bin
MAIN_PKG    := .

GO          := go
GOFLAGS    ?=
LDFLAGS    := -ldflags "-s -w"

# Default config path used when running locally
CONFIG_FILE ?= config.yaml

.PHONY: all build clean run test lint fmt vet tidy install

## all: build the binary (default target)
all: build

## build: compile the portwatch binary into ./bin/
build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PKG)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

## run: build and run portwatch with the local config file
run: build
	$(BUILD_DIR)/$(BINARY_NAME) -config $(CONFIG_FILE)

## install: install portwatch into GOPATH/bin
install:
	$(GO) install $(GOFLAGS) $(LDFLAGS) $(MAIN_PKG)

## test: run all unit tests with race detector
test:
	$(GO) test -race -count=1 ./...

## test-verbose: run tests with verbose output
test-verbose:
	$(GO) test -race -v -count=1 ./...

## lint: run golangci-lint (must be installed separately)
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found — install via: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	}
	golangci-lint run ./...

## fmt: format all Go source files
fmt:
	$(GO) fmt ./...

## vet: run go vet on all packages
vet:
	$(GO) vet ./...

## tidy: tidy and verify go modules
tidy:
	$(GO) mod tidy
	$(GO) mod verify

## clean: remove build artifacts
clean:
	@rm -rf $(BUILD_DIR)
	@echo "Cleaned $(BUILD_DIR)"

## help: print this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
