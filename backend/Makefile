SHELL := /bin/sh
.SHELLFLAGS := -eu -c
.DEFAULT_GOAL := help

GO ?= go
GOLANGCI_LINT ?= golangci-lint
GOVULNCHECK ?= govulncheck
GOLANGCI_LINT_VERSION := v2.13.1
GOVULNCHECK_VERSION := v1.7.0

# Explicit patterns fail visibly on build errors; no parse-time go list pipeline.
# HTTP integration tests live exclusively in apih-tests.
UNIT_PACKAGES := ./cmd/... ./internal/... ./pkg/...
ALL_PACKAGES := $(UNIT_PACKAGES)
COVER_PACKAGES := ./cmd/...,./internal/...,./pkg/...
COVERAGE_FILE := .coverage/unit/coverage.out

goversion ?= $(shell awk '$$1 == "go" { print $$2; exit }' go.mod)
DATABASE_URL ?= postgres://postgres:postgres@localhost:15432/changes
APIH ?= apih
API_TEST_PORT ?= 19080
export GO APIH API_TEST_PORT DATABASE_URL

.PHONY: check init format format-check lint vet test race coverage coverage-html \
        benchmark deps-audit tooling-test help test_version import-db api-test

check: format-check lint vet race tooling-test ## Run read-only formatting, lint, vet, race and tooling checks

init: ## Install pinned development tools (without changing go.mod)
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	$(GO) install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

format: ## Apply gofumpt and goimports formatting to backend Go files
	$(GOLANGCI_LINT) fmt --no-config --enable gofumpt --enable goimports $(ALL_PACKAGES)

format-check: ## Report formatting differences without rewriting files
	$(GOLANGCI_LINT) fmt --no-config --enable gofumpt --enable goimports --diff $(ALL_PACKAGES)

lint: ## Run standard golangci-lint checks plus revive (includes staticcheck)
	$(GOLANGCI_LINT) run --no-config --default standard --enable revive --timeout 5m $(ALL_PACKAGES)

vet: ## Vet all backend Go packages
	$(GO) vet $(ALL_PACKAGES)

test: ## Run uncached unit tests without database or HTTP integration tests
	$(GO) test -short -count=1 $(UNIT_PACKAGES)

race: ## Run uncached unit tests with the race detector
	$(GO) test -short -count=1 -race $(UNIT_PACKAGES)

coverage: ## Measure fresh production unit statements and enforce at least 95%
	python3 -B scripts/coverage.py

coverage-html: ## Run unit coverage and generate HTML even for a valid below-threshold profile
	python3 -B scripts/coverage.py --html

benchmark: ## Run benchmarks without ordinary or database integration tests
	$(GO) test -short -run='^$$' -bench=. -benchmem $(UNIT_PACKAGES)

deps-audit: ## Check backend dependencies for known vulnerabilities
	$(GOVULNCHECK) $(ALL_PACKAGES)

tooling-test: ## Test Makefile behavior with isolated tool and service fixtures
	python3 -B -m unittest discover -s scripts -p '*_test.py' -v
	$(GO) test -count=1 ./scripts/validate-apih-suite

import-db: ## Initialize and seed DATABASE_URL (destructive; explicit target only)
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f ../db/init.sql
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f ../db/seed.sql
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f ../db/seed-demo.sql

api-test: ## Run the HTTP-only suite and enforce 90% integration statements (no DB setup)
	python3 -B scripts/api_coverage.py


help: ## Display available backend targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "%-20s %s\n", $$1, $$2}' Makefile

test_version: ## Run checks in Docker using go.mod's Go version; override with goversion=X
	docker run --rm -v "$(CURDIR):/project" -w /project "golang:$(goversion)" sh -eu -c 'command -v python3 >/dev/null || { apt-get update && apt-get install -y --no-install-recommends python3; }; go version; make init check'
