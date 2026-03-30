package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/schigh/health/v2"
)

type mockPinger struct {
	err   error
	delay time.Duration
}

func (m *mockPinger) PingContext(ctx context.Context) error {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return m.err
}

func TestNewChecker_Defaults(t *testing.T) {
	c := NewChecker("test", &mockPinger{})
	if c.name != "test" {
		t.Fatalf("expected name %q, got %q", "test", c.name)
	}
	if c.timeout != DefaultPingTimeout {
		t.Fatalf("expected default timeout %v, got %v", DefaultPingTimeout, c.timeout)
	}
}

func TestNewChecker_WithTimeout(t *testing.T) {
	c := NewChecker("test", &mockPinger{}, WithTimeout(5*time.Second))
	if c.timeout != 5*time.Second {
		t.Fatalf("expected timeout 5s, got %v", c.timeout)
	}
}

func TestCheck_Healthy(t *testing.T) {
	c := NewChecker("db", &mockPinger{})
	result := c.Check(context.Background())

	if result.Name != "db" {
		t.Errorf("expected name %q, got %q", "db", result.Name)
	}
	if result.Status != health.StatusHealthy {
		t.Errorf("expected healthy, got %v", result.Status)
	}
	if result.Error != nil {
		t.Errorf("expected no error, got %v", result.Error)
	}
	if result.Duration == 0 {
		t.Error("expected non-zero duration")
	}
}

func TestCheck_PingError(t *testing.T) {
	pingErr := errors.New("connection refused")
	c := NewChecker("db", &mockPinger{err: pingErr})
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Errorf("expected unhealthy, got %v", result.Status)
	}
	if !errors.Is(result.Error, pingErr) {
		t.Errorf("expected %v, got %v", pingErr, result.Error)
	}
	if result.ErrorSince.IsZero() {
		t.Error("expected ErrorSince to be set")
	}
}

func TestCheck_NilPinger(t *testing.T) {
	c := NewChecker("db", nil)
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Errorf("expected unhealthy, got %v", result.Status)
	}
	if result.Error == nil || result.Error.Error() != "invalid pinger" {
		t.Errorf("expected 'invalid pinger' error, got %v", result.Error)
	}
}

func TestCheck_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := NewChecker("db", &mockPinger{})
	result := c.Check(ctx)

	if result.Status != health.StatusUnhealthy {
		t.Errorf("expected unhealthy, got %v", result.Status)
	}
	if result.Error == nil || result.Error.Error() != "invalid context" {
		t.Errorf("expected 'invalid context' error, got %v", result.Error)
	}
}

func TestCheck_Timeout(t *testing.T) {
	c := NewChecker("db", &mockPinger{delay: time.Second}, WithTimeout(time.Millisecond))
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Errorf("expected unhealthy, got %v", result.Status)
	}
	if !errors.Is(result.Error, context.DeadlineExceeded) {
		t.Errorf("expected deadline exceeded, got %v", result.Error)
	}
}
