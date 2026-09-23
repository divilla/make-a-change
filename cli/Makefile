PKG := "mch"
PKG_LIST := $(shell go list ${PKG}/... | grep -v '/api-tests' | grep -v '/integration')

.DEFAULT_GOAL := check
check: lint vet race integration-test terminal-test ## Check project

init:
	@go install golang.org/x/lint/golint@latest
	@go install honnef.co/go/tools/cmd/staticcheck@latest

lint: ## Lint the files
	@goimports -w .
	@staticcheck ${PKG_LIST}
	@golint -set_exit_status ${PKG_LIST}

vet: ## Vet the files
	@go vet ${PKG_LIST}

test: ## Run tests
	@go test -short ${PKG_LIST}

race: ## Run tests with data race detector
	@go test -race ${PKG_LIST}

integration-test: ## Run CLI and default Flow program integration tests
	@go test ./integration

terminal-test: ## Run CLI pseudo-terminal integration tests
	@go test ./integration/terminal/...

benchmark: ## Run benchmarks
	@go test -run="-" -benchmem -bench=".*" ${PKG_LIST}

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

goversion ?= "1.25"
test_version: ## Run tests inside Docker with given version (defaults to 1.25 oldest supported). Example: make test_version goversion=1.25
	@docker run --rm -it -v $(shell pwd):/project golang:$(goversion) /bin/sh -c "cd /project && make init check"
