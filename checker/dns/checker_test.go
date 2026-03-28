package dns_test

import (
	"context"
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

func TestChecker_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := dns.NewChecker("test", "localhost")
	result := c.Check(ctx)

	if result.Status != health.StatusUnhealthy {
		t.Fatalf("expected unhealthy on cancelled context, got %s", result.Status)
	}
}
