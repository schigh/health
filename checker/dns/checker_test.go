package dns_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/schigh/health/v2"
	"github.com/schigh/health/v2/checker/dns"
)

func TestChecker_Healthy(t *testing.T) {
	c := dns.NewChecker("test", "localhost")
	result := c.Check(context.Background())

	if result.Status != health.StatusHealthy {
		t.Fatalf("expected healthy, got %s (err: %v)", result.Status, result.Error)
	}
}

func TestChecker_NXDOMAIN(t *testing.T) {
	c := dns.NewChecker("test", "this.domain.does.not.exist.invalid.", dns.WithTimeout(2*time.Second))
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Fatalf("expected unhealthy for NXDOMAIN, got %s", result.Status)
	}
}

func TestChecker_ResolverError(t *testing.T) {
	// Custom resolver that always fails
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(_ context.Context, _, _ string) (net.Conn, error) {
			return nil, &net.OpError{Op: "dial", Err: fmt.Errorf("forced failure")}
		},
	}
	c := dns.NewChecker("test", "example.com",
		dns.WithResolver(resolver),
		dns.WithTimeout(time.Second),
	)
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Fatalf("expected unhealthy on resolver error, got %s", result.Status)
	}
}
