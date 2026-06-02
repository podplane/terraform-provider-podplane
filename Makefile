# Podplane <https://podplane.dev>
# Copyright 2026 Nadrama Pty Ltd
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := help

BINDIR=bin
BINARY_NAME=terraform-provider-podplane
MAIN_PKG=.

VERSION_TAG=$(shell if git diff --quiet && git diff --cached --quiet; then git describe --tags --exact-match 2>/dev/null; fi)
BUILD_VERSION=$(if $(VERSION_TAG),$(patsubst v%,%,$(VERSION_TAG)),dev)

# Cross-compilation settings, defaulting OS/ARCH to the current platform.
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
CGO_ENABLED ?= 0

.PHONY: help setup fmt lint precommit test build clean

help: ## Show available targets
	@echo "Usage: make <target>"
	@awk 'BEGIN {FS = ":.*?## "} /^##@/ {printf "\n\033[1m%s\033[0m\n", substr($$0, 5)} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Verify required tools and install the pre-commit hook
	@command -v go >/dev/null 2>&1 || { echo "go is required but not installed"; exit 1; }
	@command -v git >/dev/null 2>&1 || { echo "git is required but not installed"; exit 1; }
	@echo "All required tools are installed."
	@mkdir -p .git/hooks
	@printf '%s\n' '#!/bin/sh' 'exec make precommit' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Git pre-commit hook installed."

##@ Build & Test

fmt: ## Format Go source files
	@go fmt ./...

lint: ## Run linters
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint is required but not installed"; exit 1; }
	@golangci-lint run

precommit: ## Check formatting and run linters (read-only)
	@echo "Checking formatting..."
	@UNFORMATTED=$$(gofmt -l . 2>&1); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "The following files need formatting (run 'make fmt'):"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	fi
	@$(MAKE) lint

test: ## Run tests with race detector
	go test -v -race ./...

build: ## Build the provider binary
	mkdir -p $(BINDIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build -trimpath \
			-o $(BINDIR)/$(BINARY_NAME)_v$(BUILD_VERSION) \
			-ldflags "-X main.providerVersion=$(BUILD_VERSION)" \
			$(MAIN_PKG)

clean: ## Remove build artifacts
	rm -rf $(BINDIR)
