# health

A zero-dependency health check library for Go services. Built for Kubernetes, useful everywhere.

```go
go get github.com/schigh/health/v2
```

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

Endpoints: `/health/live`, `/health/ready`, `/health/startup`

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
    path: /health/live
    port: 8181
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health/ready
    port: 8181
  initialDelaySeconds: 5
  periodSeconds: 10

startupProbe:
  httpGet:
    path: /health/startup
    port: 8181
  failureThreshold: 30
  periodSeconds: 2
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
