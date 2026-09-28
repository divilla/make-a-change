SHELL := /bin/sh
.SHELLFLAGS := -eu -c
.DEFAULT_GOAL := help
GO ?= go
GOLANGCI_LINT ?= golangci-lint
GOVULNCHECK ?= govulncheck
GOLANGCI_LINT_VERSION := v2.13.1
GOVULNCHECK_VERSION := v1.7.0
GOLANGCI_LINT_CACHE ?= $(CURDIR)/.coverage/lint-cache
export GO GOLANGCI_LINT GOVULNCHECK GOLANGCI_LINT_CACHE
goversion ?= $(shell awk '$$1 == "go" { print $$2; exit }' go.mod)

.PHONY: help init format format-check lint vet test race check architecture coverage coverage-html integration-test terminal-test integration-coverage deps-audit tooling-test benchmark test_version

help: ## Display CLI verification commands (default)
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "%-23s %s\n", $$1, $$2}' Makefile

init: ## Install pinned formatter/linter and vulnerability scanner
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	$(GO) install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

format: ## Explicitly apply gofumpt/goimports to all Go source including tests
	python3 -B scripts/verify.py format

format-check: ## Check formatting without writing source
	python3 -B scripts/verify.py format-check

lint: ## Read-only standard checks plus revive, including test source
	python3 -B scripts/verify.py lint

vet: ## Vet all source including integration tests
	python3 -B scripts/verify.py vet

test: ## Run uncached short unit tests across the audited production inventory
	python3 -B scripts/verify.py test

race: ## Run uncached unit tests with race detection
	python3 -B scripts/verify.py race

check: ## Run all static, unit race, architecture and tooling checks, retaining failures
	$(MAKE) -k format-check lint vet race architecture tooling-test

architecture: ## Check actual module import boundaries separately from terminal coverage
	$(GO) test -count=1 ./integration -run '^TestCLIPackageBoundaries$$'

coverage: ## Fresh independent unit statements with a strict >95% gate
	python3 -B scripts/coverage.py

coverage-html: ## Unit coverage plus HTML, including valid below-target profiles
	python3 -B scripts/coverage.py --html

integration-test: ## Complete-program and standalone startup checks (no coverage claim)
	$(GO) test -count=1 -timeout=3m ./integration -skip '^TestCLIPackageBoundaries'

terminal-test: ## Real PTY checks; missing socat fails visibly
	$(GO) test -count=1 -timeout=2m -v ./integration/terminal

integration-coverage: ## Independent complete-program and covered-child PTY campaign; strict >90%
	python3 -B scripts/coverage.py --integration

deps-audit: ## Scan dependencies and report all failures
	python3 -B scripts/verify.py deps-audit

tooling-test: ## Isolated tooling, coverage accounting and harness tests
	python3 -B -m unittest discover -s scripts -p '*_test.py' -v
	$(GO) test -count=1 ./integration -run '^TestCLIPackageBoundariesFixtures$$'
	$(GO) test -count=1 ./integration/terminal -run '^TestHarness'

benchmark: ## Run benchmarks without ordinary tests
	python3 -B scripts/verify.py benchmark

test_version: ## Docker checks using go.mod version; override with goversion=X
	docker run --rm -v "$(CURDIR)/..:/project" -w /project/cli "golang:$(goversion)" sh -eu -c 'command -v python3 >/dev/null || { apt-get update && apt-get install -y --no-install-recommends python3; }; go version; make init check'
