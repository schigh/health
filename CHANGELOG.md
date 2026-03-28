# Changelog

All notable changes to this project will be documented in this file.

## [2.3.0.0] - 2026-03-28

### Added
- OpenTelemetry metrics reporter (`reporter/otel`) as separate module
  - Metrics: health.check.status, health.check.duration, health.check.executions, health.liveness/readiness/startup
  - Attributes: check name, group, component_type
- Prometheus metrics reporter (`reporter/prometheus`) as separate module
  - Metrics: health_check_status, health_check_duration_milliseconds, health_check_executions_total, health_liveness/readiness/startup
  - Labels: check, group, component_type, status
  - Configurable namespace prefix, dedicated registry, Handler() for /metrics
- gRPC reporter tests (overall health, per-check, Watch stream)
- README updated with OTel and Prometheus reporter documentation

## [2.2.0.0] - 2026-03-28

### Added
- Cached checker decorator: `health.WithCache(checker, ttl)` with double-checked locking
- HTTP reporter functional options: `httpserver.New(WithPort(...), WithMiddleware(...))`
- `BasicAuth` middleware with constant-time comparison
- gRPC health reporter (`reporter/grpc`) as separate module, implements `grpc.health.v1.Health`
- README rewrite with feature matrix, comparison table, K8s probe YAML, architecture diagram

### Changed
- HTTP reporter `Recover` middleware moved to outermost position (always catches panics)
- gRPC reporter uses `net.Listen` + `Serve` pattern (same as HTTP reporter)

## [2.1.0.0] - 2026-03-28

### Added
- 5 built-in checkers: `checker/http`, `checker/tcp`, `checker/dns`, `checker/redis`, `checker/command`
- `Group` and `ComponentType` as first-class fields on `CheckResult`
- `WithGroup()` and `WithComponentType()` functional options
- `WithLivenessImpact()` and `WithReadinessImpact()` options (replaces `WithCheckImpact`)
- Self-describing JSON health endpoints with `duration`, `lastCheck`, `group`, `componentType` fields
- Panic recovery in manager dispatch (via `safeCheck()`) and httpserver `cacheHealthChecks`
- Nil guard in `processHealthCheck` for broken checker implementations
- Redis checker uses raw RESP protocol (zero external dependencies)
- Command checker runs arbitrary `func(ctx) error` with panic recovery

### Changed
- `WithCheckImpact(bool, bool)` replaced by `WithLivenessImpact()`, `WithReadinessImpact()`
- `WithStartupImpact()` no longer takes a bool argument
- Manager now sets `CheckResult.Name` from registered name in all dispatch paths

### Fixed
- `CheckResult.Name` was not set by manager dispatch functions
- Panicking checker could crash manager processing goroutine

## [2.0.0.0] - 2026-03-28

### Added
- Native Go types (`CheckResult`, `Status` enum) replacing protobuf dependency
- Startup probe support (`SetStartup`, `WithStartupImpact`, `/health/startup` endpoint)
- Degraded health state (`StatusDegraded`) that reports without failing probes
- Generic thread-safe `internal/syncmap.Map[K,V]` replacing code-generated carto maps
- `DefaultLogger()` returning `slog.Default()` for structured logging
- New test infrastructure: startup probe and degraded state tests

### Changed
- `Checker` interface returns `*CheckResult` instead of `*v1.Check`
- `Reporter` interface gains `SetStartup` method and uses `*CheckResult`
- `CheckerFunc` signature updated for `*CheckResult`
- Manager uses buffered `checkFunnel` channel (prevents checker goroutine blocking)
- Manager uses simplified all-checks-ran detection (size comparison vs sorted key matching)
- HTTP reporter uses `net.Listen` + `http.Serve` instead of `time.Sleep` startup hack
- All logger calls converted from printf-style to slog-style key-value pairs
- Manager `Stop()` uses `errors.Join` for multi-error aggregation
- CI workflow updated to Go 1.22/1.23 matrix with actions v4/v5
- Makefile simplified for v2 (removed protobuf and code generation targets)
- golangci-lint config updated to Go 1.22, removed deprecated `exportloopref`

### Removed
- Protobuf dependency (`google.golang.org/protobuf`)
- GoMock dependency (`go.uber.org/mock`)
- xerrors dependency (`golang.org/x/xerrors`)
- Code-generated carto maps (replaced by generic syncmap)
- Proto source files and generated Go files
- Tools directory and stale tool binaries
- Root-level `config.go` (unused envconfig struct)
