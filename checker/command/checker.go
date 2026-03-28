package command

import (
	"context"
	"fmt"
	"time"

	"github.com/schigh/health/v2"
)

// Checker performs health checks by running an arbitrary function.
// This is useful for dependencies that don't have a dedicated checker.
type Checker struct {
	name string
	fn   func(ctx context.Context) error
}

// NewChecker returns a health checker that runs the provided function.
// If the function returns nil, the check is healthy. If it returns an error,
// the check is unhealthy. If it panics, the panic is recovered and the check
// is reported as unhealthy.
func NewChecker(name string, fn func(ctx context.Context) error) *Checker {
	return &Checker{name: name, fn: fn}
}

func (c *Checker) Check(ctx context.Context) (result *health.CheckResult) {
	start := time.Now()

	defer func() {
		if r := recover(); r != nil {
			result = &health.CheckResult{
				Name:      c.name,
				Status:    health.StatusUnhealthy,
				Error:     fmt.Errorf("checker panicked: %v", r),
				Duration:  time.Since(start),
				Timestamp: start,
			}
		}
	}()

	err := c.fn(ctx)
	if err != nil {
		return &health.CheckResult{
			Name:      c.name,
			Status:    health.StatusUnhealthy,
			Error:     err,
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}

	return &health.CheckResult{
		Name:      c.name,
		Status:    health.StatusHealthy,
		Duration:  time.Since(start),
		Timestamp: start,
	}
}
