# Contributing

Thank you for your interest in contributing to the health library.

## Prerequisites

- Go 1.22 or later
- Docker (for E2E tests)
- [Kind](https://kind.sigs.k8s.io/) (for E2E tests)

## Repository Structure

This is a multi-module Go repository. The core module has zero external dependencies. Reporters with heavy dependencies are separate Go modules with their own `go.mod`:

```
.                          # Core module: github.com/schigh/health/v2
├── reporter/grpc/         # Separate module: depends on google.golang.org/grpc
├── reporter/otel/         # Separate module: depends on go.opentelemetry.io/otel
├── reporter/prometheus/   # Separate module: depends on prometheus/client_golang
└── e2e/                   # E2E tests (build tag: e2e)
```

## Development Setup

Clone the repository and create a Go workspace for local development:

```bash
git clone git@github.com:schigh/health.git
cd health

# Create a workspace (not committed) for multi-module development
cat > go.work <<EOF
go 1.22.0

use (
    .
    ./reporter/grpc
    ./reporter/otel
    ./reporter/prometheus
)
EOF
```

The `go.work` file is in `.gitignore` and is not committed. It allows `go build ./...` and IDE tooling to work across all modules simultaneously.

## Running Tests

```bash
# Core module unit tests (no external deps needed)
make test

# All checks (vet + test)
make check

# Run benchmarks
go test -bench=. -benchmem ./... ./reporter/httpserver/

# E2E tests (requires Docker + Kind)
make e2e
```

## E2E Tests

The E2E suite deploys three microservices to a Kind cluster with real Postgres and Redis. See [e2e/README.md](e2e/README.md) for details.

```bash
make e2e              # full cycle: cluster, build, deploy, test, teardown
make e2e-cluster      # create Kind cluster only
make e2e-build        # build Docker images
make e2e-deploy       # deploy to cluster
make e2e-test         # run tests against running cluster
make e2e-teardown     # delete cluster
```

## Adding a New Checker

1. Create a new package under `checker/` (e.g., `checker/memcached/`)
2. Implement the `health.Checker` interface
3. Use functional options for configuration
4. Return `*health.CheckResult` with `Duration`, `Timestamp`, and `Status` populated
5. All checkers must respect `context.Context` for cancellation and timeouts
6. Add tests covering: healthy, unhealthy, timeout, and connection refused cases
7. Do not import external dependencies in checkers (use interfaces or protocol-level communication)

## Adding a New Reporter

If the reporter has no external dependencies, add it under `reporter/` in the core module.

If it requires an external dependency (like a client library), create a separate Go module:

1. Create `reporter/yourreporter/go.mod` with its own module path
2. Add a `replace` directive pointing to `../..` for local development
3. Implement the `health.Reporter` interface (all 6 methods)
4. Add the module to your local `go.work` file

## Code Style

- Run `go vet ./...` before committing
- Run `go test -race ./...` before committing
- Use functional options for all configuration
- Check errors from `AddCheck`, `AddReporter`, and `Stop`
- No bare booleans in public APIs

## Commit Messages

Follow conventional commits: `feat:`, `fix:`, `test:`, `docs:`, `chore:`.
