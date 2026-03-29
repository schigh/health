// Package health provides a health check framework for Go services.
//
// The framework is built around three interfaces: [Manager], [Checker], and [Reporter].
// A Manager orchestrates health checks and dispatches results to reporters. Checkers
// perform individual health checks against dependencies (databases, caches, HTTP
// endpoints). Reporters expose health state to external observers (HTTP endpoints,
// gRPC, Prometheus, OpenTelemetry).
//
// The core module has zero external dependencies. Reporters with heavy dependencies
// (gRPC, OTel, Prometheus) are available as separate Go modules.
//
// See https://schigh.github.io/health/ for full documentation.
package health

import "context"

// Manager defines a manager of health checks for the application. A Manager
// is a running daemon that oversees all the health checks added to it. When a
// Manager has new health check information, it dispatches an update to its
// Reporter(s).
type Manager interface {
	// Run the health check manager. Invoking this will initialize all managed
	// checks and reporters. This function returns a read-only channel of errors.
	// If a non-nil error is propagated across this channel, that means the health
	// check manager has entered an unrecoverable state, and the application
	// should halt.
	Run(context.Context) <-chan error

	// Stop the manager and all included checks and reporters. Should be called
	// when an application is shutting down gracefully.
	Stop(context.Context) error

	// AddCheck will add a named health checker to the manager. By default, an
	// added check will run once immediately upon startup, and not affect
	// liveness or readiness. Options are available to set an initial check delay,
	// a check interval, and any affects on liveness or readiness. All added
	// health checks must be named uniquely. Adding a check with the same name
	// as an existing health check (case-insensitive), will overwrite the previous
	// check. Attempting to add a check after the manager is running will return
	// an error.
	AddCheck(name string, c Checker, opts ...AddCheckOption) error

	// AddReporter adds a named health reporter to the manager. Every time a
	// health check is reported, the manager will relay the update to the
	// reporters. All added health reporters must be named uniquely.
	// Adding a reporter with the same name as an existing health reporter
	// (case-insensitive), will overwrite the previous reporter. Attempting to
	// add a reporter after the manager is running will return an error.
	AddReporter(name string, r Reporter) error
}

// Reporter reports the health status of the application to a receiving output.
// The mechanism by which the Reporter sends this information is
// implementation-dependent. Some reporters, such as an HTTP server, are
// pull-based, while others, such as a stdout reporter, are push-based. Each
// reporter variant is responsible for managing the health information passed to
// it from the health Manager. A Manager may have multiple reporters, and a
// Reporter may have multiple providers. The common dialog between reporters and
// providers is a map of CheckResult items keyed by string. It is implied that
// all health checks within a system are named uniquely. A Reporter must be
// prepared to receive updates at any time and at any frequency.
type Reporter interface {
	// Run the reporter.
	Run(context.Context) error

	// Stop the reporter and release resources.
	Stop(context.Context) error

	// SetLiveness instructs the reporter to relay the liveness of the
	// application to an external observer.
	SetLiveness(context.Context, bool)

	// SetReadiness instructs the reporter to relay the readiness of the
	// application to an external observer.
	SetReadiness(context.Context, bool)

	// SetStartup instructs the reporter to relay the startup status of the
	// application to an external observer. Startup probes tell Kubernetes
	// that the application has finished initializing.
	SetStartup(context.Context, bool)

	// UpdateHealthChecks is called from the manager to update the reported
	// health checks.
	UpdateHealthChecks(context.Context, map[string]*CheckResult)
}

// Checker performs an individual health check and returns the result
// to the health manager.
type Checker interface {
	// Check runs the health check and returns a check result.
	Check(context.Context) *CheckResult
}
