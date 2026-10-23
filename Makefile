# Makefile for uncors project
# Go-based CORS proxy development and build automation

# Variables
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build
GOINSTALL := $(GO) install
LDFLAGS := -ldflags="-s -w"
COVERAGE_FILE := coverage.out
BINARY_NAME := uncors
BINARY_WINDOWS := $(BINARY_NAME).exe
GOLANGCI_LINT_VERSION := 2.15.0

# Directories and files to clean
CLEAN_FILES := $(BINARY_NAME) $(BINARY_WINDOWS) $(COVERAGE_FILE)

# Default target
.DEFAULT_GOAL := all

# Phony targets (targets that don't represent files)
.PHONY: all help clean format upgrade test test-integration test-cover build build-release install check colors playground stories

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
	@[ "$$(golangci-lint version --short)" = "$(GOLANGCI_LINT_VERSION)" ] || \
		gum style --foreground 214 "golangci-lint $$(golangci-lint version --short) differs from $(GOLANGCI_LINT_VERSION) used on CI"
	@golangci-lint run --fix
	@golangci-lint run --fix --build-tags release

## format-docs: Format documentation files using Prettier
format-docs:
	@gum style --foreground 11 "Formatting documentation for fakedata tool..."
	@npx -y prettier ./docs/**/*.md --write
	@npx -y prettier ./*.md --write

## upgrade: Upgrade dependencies and tidy go.mod
upgrade:
	@gum style --foreground 11 "Upgrading dependencies..."
	@go-mod-upgrade
	@$(GO) mod tidy
	@gum style --foreground 11 "Running full build after upgrade..."
	@$(MAKE) all

## test: Run all tests
test:
	@gum style --foreground 11 "Running tests..."
	@$(GOTEST) -race ./...

## test-integration: Run end-to-end integration tests (real sockets + TLS)
test-integration:
	@gum style --foreground 11 "Running integration tests..."
	@$(GOTEST) -tags integration -race ./tests/integration/...

test-release:
	@gum style --foreground 11 "Running release tests..."
	@$(GOTEST) -tags release  -race ./...

## test-cover: Run tests with race detection and generate coverage report
test-cover:
	@gum style --foreground 11 "Running tests with coverage..."
	@$(GOTEST) -tags release -timeout 1m -race -v -coverprofile=$(COVERAGE_FILE) ./...
	@printf "$$(gum style --foreground 10 'Coverage report saved to') $$(gum style --foreground 190 '$(COVERAGE_FILE)')\n"

# TUI stories (testing/stories)
STORY ?=

## playground: Pick a TUI story and show it, rebuilding on every Go change (STORY= skips the picker)
playground:
	@$(GO) run ./tools/tuiplay $(if $(STORY),-story $(STORY))

## stories: Update the TUI story snapshots
stories:
	@gum style --foreground 11 "Updating story snapshots..."
	@UPDATE_SNAPS=true $(GOTEST) ./testing/stories/ -run TestStories

## build: Build all packages
build:
	@gum style --foreground 11 "Building packages..."
	@$(GOBUILD) ./...

## build-release: Build release binary with optimizations
build-release:
	@printf "$$(gum style --foreground 11 'Building release binary')\n"
	@$(GOBUILD) -tags release $(LDFLAGS) .

## install: Install the binary to GOPATH/bin
install:
	@printf "$$(gum style --foreground 11 'Installing binary')\n"
	@$(GOINSTALL) -tags release $(LDFLAGS) .

## clean: Remove generated files and binaries
clean:
	@gum style --foreground 11 "Cleaning generated files..."
	@rm -rf $(CLEAN_FILES)

## check: Run format, test, and build (quality checks)
check: format test build
	@gum style --bold --foreground 10 "✓ All checks passed!"
