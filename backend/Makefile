SHELL := /bin/sh
.SHELLFLAGS := -eu -c
.DEFAULT_GOAL := help

GO ?= go
GOLANGCI_LINT ?= golangci-lint
GOVULNCHECK ?= govulncheck
GOLANGCI_LINT_VERSION := v2.13.1
GOVULNCHECK_VERSION := v1.7.0

# Explicit patterns fail visibly on build errors; no parse-time go list pipeline.
# The HTTP test client is a test harness, not production coverage.
UNIT_PACKAGES := ./cmd/... ./internal/... ./pkg/...
ALL_PACKAGES := $(UNIT_PACKAGES) ./api-tests/...
COVER_PACKAGES := ./cmd/...,./internal/...,./pkg/...
COVERAGE_FILE := .coverage/unit/coverage.out

goversion ?= $(shell awk '$$1 == "go" { print $$2; exit }' go.mod)
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/changes
API_TEST_DB_URL ?= postgres://postgres:postgres@localhost:5432/changes_test
API_TEST_PORT ?= 19080
API_TEST_BASE_URL ?= http://localhost:$(API_TEST_PORT)
export API_TEST_DB_URL API_TEST_PORT API_TEST_BASE_URL GO

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

vet: ## Vet all backend packages, including the HTTP test harness
	$(GO) vet $(ALL_PACKAGES)

test: ## Run uncached unit tests without database or HTTP integration tests
	$(GO) test -short -count=1 $(UNIT_PACKAGES)

race: ## Run uncached unit tests with the race detector
	$(GO) test -short -count=1 -race $(UNIT_PACKAGES)

coverage: ## Measure unit coverage for every production package (target: above 95%)
	mkdir -p .coverage/unit
	rm -f "$(COVERAGE_FILE)" .coverage/unit/coverage.html
	$(GO) test -short -count=1 -race -covermode=atomic -coverpkg=$(COVER_PACKAGES) -coverprofile="$(COVERAGE_FILE)" $(UNIT_PACKAGES)
	$(GO) tool cover -func="$(COVERAGE_FILE)"

coverage-html: coverage ## Generate an HTML report after a successful unit coverage run
	$(GO) tool cover -html="$(COVERAGE_FILE)" -o .coverage/unit/coverage.html

benchmark: ## Run benchmarks without ordinary or database integration tests
	$(GO) test -short -run='^$$' -bench=. -benchmem $(UNIT_PACKAGES)

deps-audit: ## Check backend dependencies for known vulnerabilities
	$(GOVULNCHECK) $(ALL_PACKAGES)

tooling-test: ## Test Makefile behavior with isolated tool and service fixtures
	python3 -B -m unittest discover -s scripts -p '*_test.py' -v

import-db: ## Initialize and seed DATABASE_URL (destructive; explicit target only)
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f ../db/init.sql
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f ../db/seed.sql
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f ../db/seed-demo.sql

# Retain the existing Go HTTP tests until P0 introduces the APIHydra runner.
# This target does not measure the future APIHydra integration coverage gate.
api-test: ## Rebuild changes_test and run the existing Go HTTP integration suite
	bash scripts/run-api-tests.sh

help: ## Display available backend targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "%-20s %s\n", $$1, $$2}' Makefile

test_version: ## Run checks in Docker using go.mod's Go version; override with goversion=X
	docker run --rm -v "$(CURDIR):/project" -w /project "golang:$(goversion)" sh -eu -c 'command -v python3 >/dev/null || { apt-get update && apt-get install -y --no-install-recommends python3; }; go version; make init check'
