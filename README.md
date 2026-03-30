[![Test](https://github.com/schigh/health/actions/workflows/test.yaml/badge.svg)](https://github.com/schigh/health/actions/workflows/test.yaml)
[![codecov](https://codecov.io/github/schigh/health/graph/badge.svg?token=LUYRVETK3N)](https://codecov.io/github/schigh/health)
[![Go Report Card](https://goreportcard.com/badge/github.com/schigh/health/v2)](https://goreportcard.com/report/github.com/schigh/health/v2)
[![Go Reference](https://pkg.go.dev/badge/github.com/schigh/health/v2.svg)](https://pkg.go.dev/github.com/schigh/health/v2)

# health

A zero-dependency health check library for Go services. Built for Kubernetes, useful everywhere.

**[Full Documentation](https://schigh.github.io/health/)** | [pkg.go.dev](https://pkg.go.dev/github.com/schigh/health/v2)

```go
go get github.com/schigh/health/v2
```

## When to use this library

This library is designed for Go services running in Kubernetes with multiple external dependencies (databases, caches, other services). It is most valuable when:

- You need **readiness separate from liveness** (your pod is alive but Postgres is down, so you should stop receiving traffic without being killed)
- You have **startup sequencing** requirements (loading data, warming caches, waiting for dependencies before accepting traffic)
- You want **structured observability** into why a service is unhealthy, not just that it restarted
- You run **multiple services** that depend on each other and want dependency graph visibility

If your service is stateless with no external dependencies, a simple `http.HandleFunc("/healthz", ...)` returning 200 is sufficient. You don't need this library for that.

## Why this library?

| | health/v2 | heptiolabs | alexliesenfeld | InVisionApp |
|---|---|---|---|---|
| External deps | **0** | 2 | 3 | 5 |
| K8s probes | liveness, readiness, **startup** | liveness, readiness | liveness, readiness | liveness, readiness |
| Degraded state | **yes** | no | no | no |
| Built-in checkers | HTTP, TCP, DNS, Redis, DB, command | HTTP, TCP, DNS | none | none |
| Maintained | **active** | archived | active | archived |

## Quick Start

```go
package main

import (
    "context"
    "os/signal"
    "syscall"
    "time"

    "github.com/schigh/health/v2"
    "github.com/schigh/health/v2/manager/std"
    "github.com/schigh/health/v2/checker/http"
    "github.com/schigh/health/v2/checker/tcp"
    "github.com/schigh/health/v2/reporter/httpserver"
)

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    mgr := std.Manager{}

    // HTTP dependency
    mgr.AddCheck("api", http.NewChecker("api", "https://api.example.com/health"),
        health.WithCheckFrequency(health.CheckAtInterval, 10*time.Second, 0),
        health.WithLivenessImpact(),
        health.WithReadinessImpact(),
        health.WithGroup("external"),
        health.WithComponentType("http"),
    )

    // Database (TCP)
    mgr.AddCheck("postgres", tcp.NewChecker("postgres", "localhost:5432"),
        health.WithCheckFrequency(health.CheckAtInterval, 5*time.Second, 0),
        health.WithLivenessImpact(),
        health.WithReadinessImpact(),
        health.WithGroup("database"),
        health.WithComponentType("datastore"),
    )

    // HTTP reporter with BasicAuth
    mgr.AddReporter("http", httpserver.New(
        httpserver.WithPort(8181),
        httpserver.WithMiddleware(httpserver.BasicAuth("admin", "secret")),
    ))

    errChan := mgr.Run(ctx)
    select {
    case err := <-errChan:
        panic(err)
    case <-ctx.Done():
        mgr.Stop(ctx)
    }
}
```

## Built-in Checkers

All checkers are zero-dependency, using only the standard library.

| Package | What it checks | Options |
|---|---|---|
| `checker/http` | HTTP endpoint returns expected status | `WithTimeout`, `WithExpectedStatus`, `WithMethod`, `WithClient` |
| `checker/tcp` | TCP port is accepting connections | `WithTimeout` |
| `checker/dns` | Hostname resolves to an address | `WithTimeout`, `WithResolver` |
| `checker/redis` | Redis PING via raw RESP protocol | `WithTimeout`, `WithPassword` |
| `checker/db` | Database ping via `sql.DB` interface | `WithTimeout` |
| `checker/command` | Run any `func(ctx) error` | (none) |

```go
// Custom check with the command checker
mgr.AddCheck("s3", command.NewChecker("s3", func(ctx context.Context) error {
    _, err := s3Client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &bucket})
    return err
}))
```

## Caching

Wrap any checker with TTL-based caching to avoid hammering expensive dependencies:

```go
cached := health.WithCache(
    redis.NewChecker("redis", "localhost:6379"),
    30*time.Second,
)
mgr.AddCheck("redis", cached, ...)
```

## Check Metadata

Checks carry structured metadata for observability and dependency mapping:

```go
mgr.AddCheck("postgres", dbChecker,
    health.WithGroup("database"),          // logical group
    health.WithComponentType("datastore"), // component type hint
)
```

The HTTP reporter includes this metadata in the JSON response:

```json
{
  "postgres": {
    "name": "postgres",
    "status": "healthy",
    "group": "database",
    "componentType": "datastore",
    "duration": "1.234ms",
    "lastCheck": "2026-03-28T14:30:00Z"
  }
}
```

## Reporters

### HTTP Server (default)

```go
// Functional options
reporter := httpserver.New(
    httpserver.WithPort(9090),
    httpserver.WithMiddleware(httpserver.BasicAuth("user", "pass")),
)

// Or struct config
reporter := httpserver.NewReporter(httpserver.Config{
    Addr: "0.0.0.0",
    Port: 8181,
})
```

Endpoints: `/livez`, `/readyz`, `/healthz`, `/.well-known/health`

Individual checks by name ([K8s convention](https://kubernetes.io/docs/reference/using-api/health-checks/#individual-health-checks)):

```bash
curl localhost:8181/livez/postgres    # [+]postgres ok (200)
curl localhost:8181/readyz/redis      # [-]redis failed: timeout (503)
curl localhost:8181/livez?verbose     # list all checks
curl "localhost:8181/livez?verbose&exclude=redis"  # exclude a check
```

### gRPC

Implements the standard `grpc.health.v1.Health` protocol. Separate module to keep the core zero-dep.

```go
go get github.com/schigh/health/v2/reporter/grpc
```

```go
reporter := grpc.NewReporter(grpc.Config{
    Addr: "0.0.0.0:8182",
})
```

### OpenTelemetry

Emits health metrics via the OTel API. Separate module.

```
go get github.com/schigh/health/v2/reporter/otel
```

```go
reporter, err := otel.NewReporter(otel.Config{
    MeterProvider: provider, // your OTel MeterProvider
})
```

Metrics: `health.check.status`, `health.check.duration`, `health.check.executions`, `health.liveness`, `health.readiness`, `health.startup`.

### Prometheus

Exposes health metrics for Prometheus scraping. Separate module.

```
go get github.com/schigh/health/v2/reporter/prometheus
```

```go
reporter := prometheus.NewReporter(prometheus.Config{
    Namespace: "myapp", // optional prefix
})
http.Handle("/metrics", reporter.Handler())
```

Metrics: `health_check_status`, `health_check_duration_milliseconds`, `health_check_executions_total`, `health_liveness`, `health_readiness`, `health_startup`.

### stdout

Prints an ASCII table to stdout. Useful for local development.

### test

Instrumented reporter for unit tests. Tracks state changes, toggle counts, and health check updates.

## Kubernetes

```yaml
livenessProbe:
  httpGet:
    path: /livez
    port: 8181
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /readyz
    port: 8181
  initialDelaySeconds: 5
  periodSeconds: 10

startupProbe:
  httpGet:
    path: /healthz
    port: 8181
  failureThreshold: 30
  periodSeconds: 2
```

## Service Discovery

Every service using this library can expose a `/.well-known/health` manifest endpoint that describes its health checks, dependencies, and current state. Other services can discover this manifest and build transitive dependency graphs with zero infrastructure.

```go
// Enable the manifest endpoint
reporter := httpserver.New(
    httpserver.WithServiceName("orders-api"),
    httpserver.WithServiceVersion("1.2.3"),
)

// Declare dependencies between checks
mgr.AddCheck("payments", httpChecker,
    health.WithDependsOn("http://payments:8181"),
)
```

### Discovering the graph

```go
// Fetch a single service's manifest
manifest, _ := discovery.FetchManifest(ctx, "http://orders:8181")

// Walk the full dependency graph (BFS, follows HTTP DependsOn entries)
graph, _ := discovery.DiscoverGraph(ctx, "http://api-gateway:8181")

// Render as Mermaid or Graphviz
fmt.Println(graph.Mermaid())
fmt.Println(graph.DOT())
```

The manifest at `/.well-known/health` returns:

```json
{
  "service": "orders-api",
  "version": "1.2.3",
  "status": "pass",
  "checks": [
    {
      "name": "postgres",
      "status": "healthy",
      "group": "database",
      "componentType": "datastore",
      "duration": "1.2ms"
    },
    {
      "name": "payments",
      "status": "healthy",
      "dependsOn": ["http://payments:8181"]
    }
  ],
  "timestamp": "2026-03-28T20:00:00Z"
}
```

## Architecture

```
Checkers (HTTP, TCP, DNS, Redis, DB, command)
    │
    ▼
CheckResult{Name, Status, Group, ComponentType, Duration, ...}
    │
    ▼
Manager (evaluates fitness, tracks startup/liveness/readiness)
    │
    ▼
Reporters (HTTP server, gRPC, stdout, test)
    │
    ▼
Kubernetes probes, monitoring, dashboards
```

## Status

- `StatusHealthy` — check is passing
- `StatusDegraded` — check is passing with warnings (does not fail probes)
- `StatusUnhealthy` — check is failing

## License

Apache 2.0
