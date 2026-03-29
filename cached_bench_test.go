package health_test

import (
	"context"
	"testing"
	"time"

	"github.com/schigh/health/v2"
)

func BenchmarkCachedChecker_Hit(b *testing.B) {
	inner := health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		return &health.CheckResult{Name: "bench", Status: health.StatusHealthy}
	})
	c := health.WithCache(inner, time.Minute)

	// prime the cache
	c.Check(context.Background())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Check(context.Background())
	}
}

func BenchmarkCachedChecker_Miss(b *testing.B) {
	inner := health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		return &health.CheckResult{Name: "bench", Status: health.StatusHealthy}
	})
	// TTL of 0 means every call is a miss
	c := health.WithCache(inner, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Check(context.Background())
	}
}
