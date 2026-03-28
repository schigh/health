package tcp_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/schigh/health/v2"
	"github.com/schigh/health/v2/checker/tcp"
)

func TestChecker_Healthy(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	c := tcp.NewChecker("test", ln.Addr().String())
	result := c.Check(context.Background())

	if result.Status != health.StatusHealthy {
		t.Fatalf("expected healthy, got %s (err: %v)", result.Status, result.Error)
	}
}

func TestChecker_ConnectionRefused(t *testing.T) {
	c := tcp.NewChecker("test", "127.0.0.1:1", tcp.WithTimeout(100*time.Millisecond))
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Fatalf("expected unhealthy, got %s", result.Status)
	}
}

func TestChecker_Timeout(t *testing.T) {
	// Use a non-routable address to trigger timeout
	c := tcp.NewChecker("test", "192.0.2.1:1", tcp.WithTimeout(100*time.Millisecond))
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Fatalf("expected unhealthy on timeout, got %s", result.Status)
	}
}
