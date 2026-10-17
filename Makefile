# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := help
.PHONY: help setup precommit

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Install Git hooks
	@hooks=$$(git rev-parse --git-path hooks); \
	mkdir -p "$$hooks"; \
	cp scripts/git-hooks/pre-commit scripts/git-hooks/commit-msg "$$hooks/"; \
	chmod +x "$$hooks/pre-commit" "$$hooks/commit-msg"

precommit: ## Check publication metadata and shell syntax (read-only)
	@test -s README.md && test -s LICENSE && test -s releases.asc
	@jq -e '.version == 1 and .metadata.protocol_versions == ["6.0"]' terraform-registry-manifest.json >/dev/null
	@bash -n scripts/git-hooks/pre-commit scripts/git-hooks/commit-msg
