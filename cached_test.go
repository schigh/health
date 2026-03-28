package health_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/schigh/health/v2"
)

func TestCachedChecker_ReturnsCachedResult(t *testing.T) {
	var calls atomic.Int32
	inner := health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		calls.Add(1)
		return &health.CheckResult{Name: "test", Status: health.StatusHealthy}
	})

	c := health.WithCache(inner, 500*time.Millisecond)

	// first call hits the inner checker
	r1 := c.Check(context.Background())
	if r1.Status != health.StatusHealthy {
		t.Fatalf("expected healthy, got %s", r1.Status)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 inner call, got %d", calls.Load())
	}

	// second call within TTL returns cached
	r2 := c.Check(context.Background())
	if r2 != r1 {
		t.Fatal("expected same pointer for cached result")
	}
	if calls.Load() != 1 {
		t.Fatalf("expected still 1 inner call, got %d", calls.Load())
	}
}

func TestCachedChecker_RefreshesAfterTTL(t *testing.T) {
	var calls atomic.Int32
	inner := health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		calls.Add(1)
		return &health.CheckResult{Name: "test", Status: health.StatusHealthy}
	})

	c := health.WithCache(inner, 50*time.Millisecond)

	c.Check(context.Background())
	if calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", calls.Load())
	}

	time.Sleep(60 * time.Millisecond)

	c.Check(context.Background())
	if calls.Load() != 2 {
		t.Fatalf("expected 2 calls after TTL, got %d", calls.Load())
	}
}

func TestCachedChecker_ConcurrentAccess(t *testing.T) {
	var calls atomic.Int32
	inner := health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		calls.Add(1)
		time.Sleep(10 * time.Millisecond) // simulate slow check
		return &health.CheckResult{Name: "test", Status: health.StatusHealthy}
	})

	c := health.WithCache(inner, time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := c.Check(context.Background())
			if r.Status != health.StatusHealthy {
				t.Errorf("expected healthy, got %s", r.Status)
			}
		}()
	}
	wg.Wait()

	// with TTL=1s and 50 concurrent calls, should have very few inner calls
	// (ideally 1, but up to a few due to lock contention on first call)
	if calls.Load() > 5 {
		t.Fatalf("expected few inner calls with caching, got %d", calls.Load())
	}
}

func TestCachedChecker_FirstCallSynchronous(t *testing.T) {
	inner := health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		return &health.CheckResult{Name: "test", Status: health.StatusDegraded}
	})

	c := health.WithCache(inner, time.Second)
	r := c.Check(context.Background())

	if r.Status != health.StatusDegraded {
		t.Fatalf("expected degraded from first call, got %s", r.Status)
	}
}
