package health

import (
	"context"
	"time"
)

// CheckerFunc is a functional health checker.
type CheckerFunc func(context.Context) *CheckResult

// Check satisfies Checker.
func (cf CheckerFunc) Check(ctx context.Context) *CheckResult {
	return cf(ctx)
}

// AddCheckOptions contain the options needed to add a new health check to the manager.
type AddCheckOptions struct {
	Frequency        CheckFrequency
	Delay            time.Duration
	Interval         time.Duration
	AffectsLiveness  bool
	AffectsReadiness bool
	AffectsStartup   bool
	Group            string
	ComponentType    string
}

// AddCheckOption is a functional option for adding a Checker to a health manager.
type AddCheckOption func(*AddCheckOptions)

// CheckFrequency is a set of flags to instruct the check scheduling.
type CheckFrequency uint

const (
	// CheckOnce instructs the Checker to perform its check one time. If the
	// CheckAfter flag is set, CheckOnce will perform the check after a duration
	// specified by the desired configuration.
	CheckOnce CheckFrequency = 1 << iota

	// CheckAtInterval instructs the Checker to perform its check at a specified
	// interval. If the CheckAfter flag is set, this check will begin after a
	// lapse of the combined Delay and Interval.
	CheckAtInterval

	// CheckAfter instructs the Checker to wait until after a specified time to
	// perform its check.
	CheckAfter
)

// WithCheckFrequency tells the health instance the CheckFrequency at which it will perform check with the specified Checker
// instance. If the value for CheckFrequency is CheckOnce, the Interval parameter is ignored. If the value for
// CheckFrequency is CheckAtInterval, the value of Interval will be used. If the value of Interval is equal to or less
// than zero, then the default Interval is used. If the value of Delay is equal to or less than zero, it is ignored.
// This option is not additive, so multiple invocations of this option will result in the last invocation being used to
// configure the Checker.
func WithCheckFrequency(f CheckFrequency, interval, delay time.Duration) AddCheckOption {
	return func(o *AddCheckOptions) {
		o.Frequency = f
		o.Interval = interval
		o.Delay = delay
	}
}

// WithLivenessImpact marks a health check as affecting the liveness of the application.
// If a check that affects liveness fails, readiness is also affected.
func WithLivenessImpact() AddCheckOption {
	return func(o *AddCheckOptions) {
		o.AffectsLiveness = true
	}
}

// WithReadinessImpact marks a health check as affecting the readiness of the application.
func WithReadinessImpact() AddCheckOption {
	return func(o *AddCheckOptions) {
		o.AffectsReadiness = true
	}
}

// WithStartupImpact marks a health check as affecting startup probes. Startup checks
// must all pass before liveness and readiness probes are evaluated. Once all startup
// checks pass, startup is considered complete and is not re-evaluated.
func WithStartupImpact() AddCheckOption {
	return func(o *AddCheckOptions) {
		o.AffectsStartup = true
	}
}

// WithGroup assigns a logical group to a health check (e.g., "database", "cache", "external").
// Groups are included in self-describing health endpoints and can be used for filtering.
func WithGroup(group string) AddCheckOption {
	return func(o *AddCheckOptions) {
		o.Group = group
	}
}

// WithComponentType assigns a component type hint to a health check (e.g., "datastore", "http", "tcp").
// Component types are included in self-describing health endpoints.
func WithComponentType(ct string) AddCheckOption {
	return func(o *AddCheckOptions) {
		o.ComponentType = ct
	}
}
