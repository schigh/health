package health

import (
	"context"
	"time"

	healthpb "github.com/schigh/health/pkg/v1"
)

// CheckerFunc is a functional health checker.
type CheckerFunc func(context.Context) *healthpb.Check

// Check satisfies Checker.
func (cf CheckerFunc) Check(ctx context.Context) *healthpb.Check {
	return cf(ctx)
}

// AddCheckOptions contain the options needed to add a new health check to the manager.
type AddCheckOptions struct {
	Frequency        CheckFrequency
	Delay            time.Duration
	Interval         time.Duration
	AffectsLiveness  bool
	AffectsReadiness bool
}

// AddCheckOption is a functional option for adding a Checker to a health managers.
type AddCheckOption func(*AddCheckOptions)

// CheckFrequency is a set of flags to instruct the.
type CheckFrequency uint

const (
	// CheckOnce instructs the Checker to perform its check one time. If the
	// CheckAfter flag is set, CheckOnce will perform the check after a duration
	// specified by the desired configuration.
	CheckOnce CheckFrequency = 1 << iota

	// CheckAtInterval instructs the Checker to perform its check at a specified
	// Interval. If the CheckAfter flag is set, this check will begin after a
	// lapse of the combined Delay and Interval.
	CheckAtInterval

	// CheckAfter instructs the Checker to wait until after a specified time to
	// perform its check.
	CheckAfter
)

// WithCheckFrequency tells the health instance the CheckFrequency at which it will perform check with the specified Checker
// instance. If the value for CheckFrequency is CheckOnce, the Interval parameter is ignored.  If the value for
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

// WithCheckImpact tells the health instance that a healthcheck affects either the liveness or readiness of the application.
// Liveness and readiness are ways Kubernetes determines the fitness of a pod
// (https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/).
// If liveness is affected by the failing health check, then readiness is also affected. By default, application
// liveness and readiness are not affected by health check.
func WithCheckImpact(liveness, readiness bool) AddCheckOption {
	return func(o *AddCheckOptions) {
		o.AffectsLiveness = liveness
		o.AffectsReadiness = readiness
	}
}
