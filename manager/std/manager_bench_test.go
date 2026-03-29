package std_test

import (
	"context"
	"testing"
	"time"

	"github.com/schigh/health/v2"
	"github.com/schigh/health/v2/manager/std"
	"github.com/schigh/health/v2/reporter/test"
)

func BenchmarkManagerProcessCheck(b *testing.B) {
	mgr := &std.Manager{}
	rpt := &test.Reporter{}

	_ = mgr.AddCheck("bench", health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		return &health.CheckResult{Name: "bench", Status: health.StatusHealthy}
	}), health.WithCheckFrequency(health.CheckAtInterval, time.Hour, 0))
	_ = mgr.AddReporter("test", rpt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = mgr.Run(ctx)

	// wait for at least one check to establish baseline
	time.Sleep(50 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rpt.Report()
	}

	_ = mgr.Stop(ctx)
}
