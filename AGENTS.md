# Repository Instructions

This file provides guidance to AGENTS when working with code in this repository.

## Development Commands

The project uses a Makefile for common development tasks:

- `make check` - Run linting, vetting, and race condition tests (default target)
- `make init` - Install required linting tools (golint, staticcheck)
- `make lint` - Run goimports, staticcheck and golint
- `make vet` - Run go vet
- `make test` - Run short tests
- `make race` - Run tests with race detector
- `make benchmark` - Run benchmarks
- `make coverage` - Display test coverage

Example commands for development:
```bash
# Setup development environment
make init

# Run all checks (lint, vet, race)
make check

# Run specific tests
go test ./middleware/...
go test -race ./...

# Run benchmarks
make benchmark
```

## Unit-test requirements

- Cover every acceptance criterion with at least one unit test.
- Implement unit tests for all production code and keep unit-test coverage
  greater than 95%.
- Avoid tests that do not add coverage unless a test exists to prove a specific
  Acceptance Criteria bullet.
- Every individual Acceptance Criteria bullet must have at least one unit test
  implemented and maintained in the test suite.
