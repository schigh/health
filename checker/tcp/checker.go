package tcp

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/schigh/health/v2"
)

const DefaultTimeout = 5 * time.Second

// Checker performs TCP dial health checks.
type Checker struct {
	name    string
	addr    string
	timeout time.Duration
}

// Option is a functional option for configuring a TCP Checker.
type Option func(*Checker)

// WithTimeout sets the dial timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Checker) { c.timeout = d }
}

// NewChecker returns a TCP health checker for the given address (host:port).
func NewChecker(name, addr string, opts ...Option) *Checker {
	c := &Checker{name: name, addr: addr, timeout: DefaultTimeout}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *Checker) Check(ctx context.Context) *health.CheckResult {
	start := time.Now()

	var d net.Dialer
	d.Timeout = c.timeout

	conn, err := d.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return &health.CheckResult{
			Name:      c.name,
			Status:    health.StatusUnhealthy,
			Error:     fmt.Errorf("dial %s: %w", c.addr, err),
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}
	_ = conn.Close()

	return &health.CheckResult{
		Name:      c.name,
		Status:    health.StatusHealthy,
		Duration:  time.Since(start),
		Timestamp: start,
	}
}
