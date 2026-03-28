package dns

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/schigh/health/v2"
)

const DefaultTimeout = 5 * time.Second

// Checker performs DNS resolution health checks.
type Checker struct {
	name     string
	hostname string
	resolver *net.Resolver
	timeout  time.Duration
}

// Option is a functional option for configuring a DNS Checker.
type Option func(*Checker)

// WithTimeout sets the lookup timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Checker) { c.timeout = d }
}

// WithResolver sets a custom net.Resolver.
func WithResolver(r *net.Resolver) Option {
	return func(c *Checker) { c.resolver = r }
}

// NewChecker returns a DNS health checker that resolves the given hostname.
func NewChecker(name, hostname string, opts ...Option) *Checker {
	c := &Checker{name: name, hostname: hostname, timeout: DefaultTimeout}
	for _, o := range opts {
		o(c)
	}
	if c.resolver == nil {
		c.resolver = net.DefaultResolver
	}
	return c
}

func (c *Checker) Check(ctx context.Context) *health.CheckResult {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	addrs, err := c.resolver.LookupHost(ctx, c.hostname)
	if err != nil {
		return &health.CheckResult{
			Name:      c.name,
			Status:    health.StatusUnhealthy,
			Error:     fmt.Errorf("lookup %s: %w", c.hostname, err),
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}

	if len(addrs) == 0 {
		return &health.CheckResult{
			Name:      c.name,
			Status:    health.StatusUnhealthy,
			Error:     fmt.Errorf("lookup %s: no addresses returned", c.hostname),
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}

	return &health.CheckResult{
		Name:      c.name,
		Status:    health.StatusHealthy,
		Duration:  time.Since(start),
		Timestamp: start,
		Metadata:  map[string]string{"resolved": addrs[0]},
	}
}
