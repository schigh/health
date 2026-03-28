# Changelog

All notable changes to this project will be documented in this file.

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
