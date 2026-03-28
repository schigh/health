package db

import (
	"context"
	"errors"
	"time"

	"github.com/schigh/health"
)

const (
	DefaultPingTimeout = 3 * time.Second
)

// CtxPinger defines the interface for the database
// functionality used to perform the health check.
type CtxPinger interface {
	PingContext(context.Context) error
}

// Checker implements health.Checker.
type Checker struct {
	name    string
	pinger  CtxPinger
	timeout time.Duration
}

// Option is a functional decorator for creating a new Checker.
type Option func(*Checker)

// WithTimeout sets the timeout on the PingContext check.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Checker) {
		c.timeout = timeout
	}
}

// NewChecker returns a Checker using the provided name and CtxPinger.
func NewChecker(name string, pinger CtxPinger, opts ...Option) *Checker {
	out := Checker{
		name:    name,
		pinger:  pinger,
		timeout: DefaultPingTimeout,
	}

	for i := range opts {
		opts[i](&out)
	}

	return &out
}

func (c *Checker) Check(ctx context.Context) *health.CheckResult {
	now := time.Now()
	select {
	case <-ctx.Done():
		return &health.CheckResult{
			Name:       c.name,
			Status:     health.StatusUnhealthy,
			Error:      errors.New("invalid context"),
			ErrorSince: now,
			Timestamp:  now,
			Metadata: map[string]string{
				"origin": "github.com/schigh/health/checker/db.Checker.Check",
			},
		}
	default:
	}

	out := health.CheckResult{
		Name:      c.name,
		Status:    health.StatusHealthy,
		Timestamp: now,
	}

	if c.pinger == nil {
		out.Status = health.StatusUnhealthy
		out.Error = errors.New("invalid pinger")
		out.ErrorSince = now
		out.Metadata = map[string]string{
			"origin": "github.com/schigh/health/checker/db.Checker.Check",
		}
		return &out
	}

	cCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	start := time.Now()
	err := c.pinger.PingContext(cCtx)
	out.Duration = time.Since(start)

	if err != nil {
		out.Status = health.StatusUnhealthy
		out.Error = err
		out.ErrorSince = now
		out.Metadata = map[string]string{
			"origin": "github.com/schigh/health/checker/db.Checker.Check",
		}
	}

	return &out
}
