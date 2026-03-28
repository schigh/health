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
type CheckResult struct {
	Name             string
	Status           Status
	AffectsLiveness  bool
	AffectsReadiness bool
	AffectsStartup   bool
	Group            string
	ComponentType    string
	DependsOn        []string
	Error            error
	ErrorSince       time.Time
	Duration         time.Duration
	Metadata         map[string]string
	Timestamp        time.Time
}
