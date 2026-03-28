package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/schigh/health/v2"
	"github.com/schigh/health/v2/checker/command"
)

func TestChecker_Healthy(t *testing.T) {
	c := command.NewChecker("test", func(_ context.Context) error {
		return nil
	})
	result := c.Check(context.Background())

	if result.Status != health.StatusHealthy {
		t.Fatalf("expected healthy, got %s", result.Status)
	}
	if result.Duration == 0 {
		t.Fatal("expected non-zero duration")
	}
}

func TestChecker_Error(t *testing.T) {
	c := command.NewChecker("test", func(_ context.Context) error {
		return errors.New("something broke")
	})
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Fatalf("expected unhealthy, got %s", result.Status)
	}
	if result.Error == nil {
		t.Fatal("expected error to be set")
	}
}

func TestChecker_Panic(t *testing.T) {
	c := command.NewChecker("test", func(_ context.Context) error {
		panic("oh no")
	})
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Fatalf("expected unhealthy on panic, got %s", result.Status)
	}
	if result.Error == nil {
		t.Fatal("expected error from recovered panic")
	}
}
