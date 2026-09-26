# Repository Instructions

Follow these instructions when working on code in this repository.

## Development Commands

The project uses a Makefile for common development tasks:

- `make check` - Run linting, vetting, and tests with the race detector (default target)
- `make init` - Install required linting tools (golint, staticcheck)
- `make lint` - Run goimports, staticcheck, and golint
- `make vet` - Run `go vet`
- `make test` - Run short tests
- `make race` - Run tests with the race detector
- `make benchmark` - Run benchmarks
- `make coverage` - Display test coverage

Examples:

```bash
# Set up the development environment
make init

# Run all checks (lint, vet, race)
make check

# Run specific tests
go test ./middleware/...
go test -race ./...

# Run benchmarks
make benchmark
```

## Unit Test Requirements

- Implement and maintain at least one unit test for every acceptance criterion,
  including every individual acceptance criteria bullet.
- Write unit tests for all production code and keep unit test coverage greater
  than 95%.
- Avoid tests that do not increase coverage unless they prove a specific
  acceptance criterion bullet.

## Architecture

### Backend

The backend follows Screaming Architecture:

- `pkg/*` contains loaders and abstractions.
- Packages under `internal/*`, except `dto` and `health`, use three layers:

  1. **API (`api`)** imports `echo`. Its methods should contain only code that
     requires `echo`; move everything else to the service layer.
  2. **Service (`service`)** contains business logic and must never import
     `echo` or `pgx`.
  3. **Repository (`repo`)** imports `pgx` and contains only code that cannot
     work without `pgx`.
