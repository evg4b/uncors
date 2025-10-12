# Makefile for uncors project
# Go-based CORS proxy development and build automation

# Variables
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build
GOINSTALL := $(GO) install
VERSION := $(shell git rev-parse --short HEAD)
LDFLAGS := -ldflags="-s -w -X 'main.Version=$(VERSION)'"
COVERAGE_FILE := coverage.out
BINARY_NAME := uncors
BINARY_WINDOWS := $(BINARY_NAME).exe

# Directories and files to clean
CLEAN_FILES := $(BINARY_NAME) $(BINARY_WINDOWS) $(COVERAGE_FILE) \
	./tools/fakedata/docs.md ./tools/fakedata/scheme.json

# Default target
.DEFAULT_GOAL := all

# Phony targets (targets that don't represent files)
.PHONY: all help clean format upgrade test test-integration test-all test-cover build build-release install check colors

# Target descriptions and implementations

## all: Run the full build pipeline (clean, check, build, test)
all: clean check build test

## help: Display this help message
help:
	@gum style --bold --foreground 86 "Available targets:"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /' | column -t -s ':'

## format: Format code using gofmt, gofumpt, and golangci-lint
format:
	@gum style --foreground 11 "Formatting code..."
	@gofmt -l -s -w .
	@gofumpt -l -w .
	@golangci-lint run --fix

## upgrade: Upgrade dependencies and tidy go.mod
upgrade:
	@gum style --foreground 11 "Upgrading dependencies..."
	@go-mod-upgrade
	@$(GO) mod tidy
	@gum style --foreground 11 "Running full build after upgrade..."
	@$(MAKE) all

## test: Run all tests (unit tests only)
test:
	@gum style --foreground 11 "Running tests..."
	@$(GOTEST) -short ./...

## test-integration: Run integration tests
test-integration: build-release
	@gum style --foreground 11 "Running integration tests..."
	@$(GOTEST) -v ./tests/integration/...

## test-all: Run all tests including integration tests
test-all: test test-integration
	@gum style --bold --foreground 10 "✓ All tests passed!"

## test-cover: Run tests with race detection and generate coverage report
test-cover:
	@gum style --foreground 11 "Running tests with coverage..."
	@$(GOTEST) -tags release -timeout 1m -race -v -coverprofile=$(COVERAGE_FILE) ./...
	@printf "$$(gum style --foreground 10 'Coverage report saved to') $$(gum style --foreground 190 '$(COVERAGE_FILE)')\n"

## build: Build all packages
build:
	@gum style --foreground 11 "Building packages..."
	@$(GOBUILD) ./...

## build-release: Build release binary with optimizations
build-release:
	@printf "$$(gum style --foreground 11 'Building release binary') $$(gum style --foreground 190 '(version: $(VERSION))')\n"
	@$(GOBUILD) -tags release $(LDFLAGS) .

## install: Install the binary to GOPATH/bin
install:
	@printf "$$(gum style --foreground 11 'Installing binary') $$(gum style --foreground 190 '(version: $(VERSION))')\n"
	@$(GOINSTALL) -tags release $(LDFLAGS) .

## clean: Remove generated files and binaries
clean:
	@gum style --foreground 11 "Cleaning generated files..."
	@rm -rf $(CLEAN_FILES)

## check: Run format, test, and build (quality checks)
check: format test build
	@gum style --bold --foreground 10 "✓ All checks passed!"

## colors: Display all available gum color codes
colors:
	@echo "Gum color palette (256 colors):"
	@for i in $$(seq 0 255); do \
		gum style --foreground $$i "Color $$i: Sample text"; \
	done
