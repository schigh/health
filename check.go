package health

import "time"

// Status represents the health status of a check.
type Status int

const (
	// StatusHealthy indicates the check is passing.
	StatusHealthy Status = iota

	// StatusDegraded indicates the check is passing but with warnings.
	// Degraded checks do not fail liveness or readiness probes.
	StatusDegraded

	// StatusUnhealthy indicates the check is failing.
	StatusUnhealthy
)

// String returns the lowercase string representation of a Status.
func (s Status) String() string {
	switch s {
	case StatusHealthy:
		return "healthy"
	case StatusDegraded:
		return "degraded"
	case StatusUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// CheckResult is the outcome of a single health check execution.
//
// Some fields are set by the checker (Status, Error, Duration, Timestamp, Metadata),
// while others are overridden by the manager from the registered [AddCheckOptions]
// (Name, AffectsLiveness, AffectsReadiness, AffectsStartup, Group, ComponentType, DependsOn).
type CheckResult struct {
	// Name identifies the check. Set by the manager from the registered check name.
	Name string
	// Status is the health status of this check.
	Status Status
	// AffectsLiveness indicates whether a failing check should affect liveness. Set by manager.
	AffectsLiveness bool
	// AffectsReadiness indicates whether a failing check should affect readiness. Set by manager.
	AffectsReadiness bool
	// AffectsStartup indicates whether this check must pass before startup completes. Set by manager.
	AffectsStartup bool
	// Group is the logical group for this check (e.g., "database", "cache"). Set by manager.
	Group string
	// ComponentType is a type hint for observability tools (e.g., "datastore", "http"). Set by manager.
	ComponentType string
	// DependsOn lists service URLs this check depends on, used by the discovery protocol. Set by manager.
	DependsOn []string
	// Error is the error from the last check execution, if any. Set by checker.
	Error error
	// ErrorSince is when the error state began. Set by checker.
	ErrorSince time.Time
	// Duration is how long the check took to execute. Set by checker.
	Duration time.Duration
	// Metadata is arbitrary key-value data for observability. Set by checker.
	Metadata map[string]string
	// Timestamp is when this check result was produced. Set by checker.
	Timestamp time.Time
}
