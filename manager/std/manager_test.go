package std_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/schigh/health/v2"
	"github.com/schigh/health/v2/manager/std"
	"github.com/schigh/health/v2/reporter/test"
)

func TestManager_NoCheckers(t *testing.T) {
	mgr := &std.Manager{}
	ch := mgr.Run(context.Background())
	err := <-ch
	if err == nil || !strings.Contains(err.Error(), "there are no checkers") {
		t.Fatalf("expected 'no checkers' error, got: %v", err)
	}
}

func TestManager_NoReporters(t *testing.T) {
	mgr := &std.Manager{}
	_ = mgr.AddCheck("test", health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		return &health.CheckResult{}
	}))
	ch := mgr.Run(context.Background())
	err := <-ch
	if err == nil || !strings.Contains(err.Error(), "there are no reporters") {
		t.Fatalf("expected 'no reporters' error, got: %v", err)
	}
}

func TestManager_OneTimeCheck(t *testing.T) {
	mgr := &std.Manager{}
	rpt := &test.Reporter{}

	_ = mgr.AddCheck("test", health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		return &health.CheckResult{Name: "test", Status: health.StatusHealthy}
	}))
	_ = mgr.AddReporter("test", rpt)

	ctx, cancel := context.WithCancel(context.Background())
	_ = mgr.Run(ctx)

	// wait for the one-time check to be processed
	waitFor(t, 2*time.Second, func() bool {
		return rpt.Report().NumHealthCheckUpdates >= 1
	})

	report := rpt.Report()
	if report.NumHealthCheckUpdates < 1 {
		t.Fatalf("expected at least 1 health check update, got %d", report.NumHealthCheckUpdates)
	}
	if report.NumLivenessStateChanges < 1 {
		t.Fatal("expected at least 1 liveness state change")
	}

	cancel()
	_ = mgr.Stop(ctx)
}

func TestManager_TwoOneTimeChecks(t *testing.T) {
	mgr := &std.Manager{}
	rpt := &test.Reporter{}

	_ = mgr.AddCheck("test1", health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		return &health.CheckResult{Name: "test1", Status: health.StatusHealthy}
	}))
	_ = mgr.AddCheck("test2", health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		return &health.CheckResult{Name: "test2", Status: health.StatusHealthy}
	}))
	_ = mgr.AddReporter("test", rpt)

	ctx, cancel := context.WithCancel(context.Background())
	_ = mgr.Run(ctx)

	waitFor(t, 2*time.Second, func() bool {
		return rpt.Report().NumHealthCheckUpdates >= 2
	})

	report := rpt.Report()
	if report.NumHealthCheckUpdates < 2 {
		t.Fatalf("expected at least 2 health check updates, got %d", report.NumHealthCheckUpdates)
	}

	cancel()
	_ = mgr.Stop(ctx)
}

func TestManager_IntervalCheck(t *testing.T) {
	mgr := &std.Manager{}
	rpt := &test.Reporter{}

	_ = mgr.AddCheck("test", health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		return &health.CheckResult{Name: "test", Status: health.StatusHealthy}
	}), health.WithCheckFrequency(health.CheckAtInterval, 50*time.Millisecond, 0))
	_ = mgr.AddReporter("test", rpt)

	ctx, cancel := context.WithCancel(context.Background())
	_ = mgr.Run(ctx)

	// wait for at least 3 check cycles
	waitFor(t, 2*time.Second, func() bool {
		return rpt.Report().NumHealthCheckUpdates >= 3
	})

	report := rpt.Report()
	if report.NumHealthCheckUpdates < 3 {
		t.Fatalf("expected at least 3 health check updates, got %d", report.NumHealthCheckUpdates)
	}

	cancel()
	_ = mgr.Stop(ctx)
}

func TestManager_IntervalCheckWithDelay(t *testing.T) {
	mgr := &std.Manager{}
	rpt := &test.Reporter{}

	_ = mgr.AddCheck("test", health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		return &health.CheckResult{Name: "test", Status: health.StatusHealthy}
	}), health.WithCheckFrequency(health.CheckAtInterval|health.CheckAfter, 50*time.Millisecond, 100*time.Millisecond))
	_ = mgr.AddReporter("test", rpt)

	ctx, cancel := context.WithCancel(context.Background())
	_ = mgr.Run(ctx)

	// after delay + a few intervals, should have updates
	waitFor(t, 2*time.Second, func() bool {
		return rpt.Report().NumHealthCheckUpdates >= 2
	})

	report := rpt.Report()
	if report.NumHealthCheckUpdates < 2 {
		t.Fatalf("expected at least 2 health check updates after delay, got %d", report.NumHealthCheckUpdates)
	}

	cancel()
	_ = mgr.Stop(ctx)
}

func TestManager_LivenessAffected(t *testing.T) {
	mgr := &std.Manager{}
	rpt := &test.Reporter{}

	// checker that fails, then succeeds, then fails, then succeeds
	checker := NewMockChecker(
		&health.CheckResult{Name: "test", Status: health.StatusUnhealthy},
		&health.CheckResult{Name: "test", Status: health.StatusHealthy},
		&health.CheckResult{Name: "test", Status: health.StatusUnhealthy},
		&health.CheckResult{Name: "test", Status: health.StatusHealthy},
	)

	_ = mgr.AddCheck("test", checker,
		health.WithCheckFrequency(health.CheckAtInterval, 100*time.Millisecond, 0),
		health.WithCheckImpact(true, true),
	)
	_ = mgr.AddReporter("test", rpt)

	ctx, cancel := context.WithCancel(context.Background())
	_ = mgr.Run(ctx)

	// wait for all 4 checks to be processed
	waitFor(t, 3*time.Second, func() bool {
		return rpt.Report().NumHealthCheckUpdates >= 4
	})

	cancel()
	_ = mgr.Stop(ctx)

	time.Sleep(50 * time.Millisecond)
	report := rpt.Report()

	// liveness should have toggled multiple times:
	// initial on (1), then off from unhealthy (2), on from healthy (3), off from unhealthy (4), on from healthy (5)
	if report.NumLivenessStateChanges < 3 {
		t.Fatalf("expected at least 3 liveness state changes, got %d", report.NumLivenessStateChanges)
	}
}

func TestManager_ReadinessAffected(t *testing.T) {
	mgr := &std.Manager{}
	rpt := &test.Reporter{}

	checker := NewMockChecker(
		&health.CheckResult{Name: "test", Status: health.StatusUnhealthy},
		&health.CheckResult{Name: "test", Status: health.StatusUnhealthy},
		&health.CheckResult{Name: "test", Status: health.StatusHealthy},
	)

	_ = mgr.AddCheck("test", checker,
		health.WithCheckFrequency(health.CheckAtInterval, 100*time.Millisecond, 0),
		health.WithCheckImpact(false, true),
	)
	_ = mgr.AddReporter("test", rpt)

	ctx, cancel := context.WithCancel(context.Background())
	_ = mgr.Run(ctx)

	waitFor(t, 3*time.Second, func() bool {
		return rpt.Report().NumHealthCheckUpdates >= 3
	})

	// after 3rd check (healthy), readiness should have been set at least once
	report := rpt.Report()
	if report.NumReadinessStateChanges < 1 {
		t.Fatalf("expected at least 1 readiness state change, got %d", report.NumReadinessStateChanges)
	}

	cancel()
	_ = mgr.Stop(ctx)
}

func TestManager_NeverLive(t *testing.T) {
	mgr := &std.Manager{}
	rpt := &test.Reporter{}

	// always unhealthy, affects liveness
	_ = mgr.AddCheck("test", health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		return &health.CheckResult{Name: "test", Status: health.StatusUnhealthy}
	}),
		health.WithCheckFrequency(health.CheckAtInterval, 100*time.Millisecond, 0),
		health.WithCheckImpact(true, true),
	)
	_ = mgr.AddReporter("test", rpt)

	ctx, cancel := context.WithCancel(context.Background())
	_ = mgr.Run(ctx)

	waitFor(t, 2*time.Second, func() bool {
		return rpt.Report().NumHealthCheckUpdates >= 3
	})

	report := rpt.Report()

	// liveness should have been set false after the first check
	// (initial true from Run, then false from unhealthy check = 2 changes)
	if report.NumLivenessStateChanges < 2 {
		t.Fatalf("expected at least 2 liveness state changes, got %d", report.NumLivenessStateChanges)
	}

	// readiness should never have been set true (check is always unhealthy)
	if report.NumReadinessSetTrue > 0 {
		t.Fatal("readiness should never have been set true for always-unhealthy check")
	}

	cancel()
	_ = mgr.Stop(ctx)
}

func TestManager_StartupProbe(t *testing.T) {
	mgr := &std.Manager{}
	rpt := &test.Reporter{}

	var callCount atomic.Int32
	_ = mgr.AddCheck("startup_check",
		health.CheckerFunc(func(_ context.Context) *health.CheckResult {
			n := callCount.Add(1)
			if n <= 2 {
				return &health.CheckResult{Name: "startup_check", Status: health.StatusUnhealthy}
			}
			return &health.CheckResult{Name: "startup_check", Status: health.StatusHealthy}
		}),
		health.WithCheckFrequency(health.CheckAtInterval, 100*time.Millisecond, 0),
		health.WithCheckImpact(true, true),
		health.WithStartupImpact(true),
	)

	_ = mgr.AddReporter("test", rpt)

	ctx, cancel := context.WithCancel(context.Background())
	_ = mgr.Run(ctx)

	// wait for startup to complete (3rd check is healthy)
	waitFor(t, 3*time.Second, func() bool {
		return rpt.Report().IsStartup
	})

	report := rpt.Report()
	if !report.IsStartup {
		t.Fatal("expected startup to be true after startup check passed")
	}
	if !report.IsLive {
		t.Fatal("expected liveness to be true after startup completed")
	}

	cancel()
	_ = mgr.Stop(ctx)
}

func TestManager_DegradedCheck(t *testing.T) {
	mgr := &std.Manager{}
	rpt := &test.Reporter{}

	_ = mgr.AddCheck("degraded_check",
		health.CheckerFunc(func(_ context.Context) *health.CheckResult {
			return &health.CheckResult{Name: "degraded_check", Status: health.StatusDegraded}
		}),
		health.WithCheckImpact(true, true),
	)

	_ = mgr.AddReporter("test", rpt)

	ctx, cancel := context.WithCancel(context.Background())
	_ = mgr.Run(ctx)

	// wait for the check to be processed and readiness to be set
	waitFor(t, 2*time.Second, func() bool {
		r := rpt.Report()
		return r.NumHealthCheckUpdates >= 1 && r.IsReady
	})

	report := rpt.Report()

	// degraded should NOT fail liveness or readiness
	if !report.IsLive {
		t.Fatal("expected liveness to be true for degraded check")
	}
	if !report.IsReady {
		t.Fatal("expected readiness to be true for degraded check")
	}

	cancel()
	_ = mgr.Stop(ctx)
}

func TestManager_AddCheckWhileRunning(t *testing.T) {
	mgr := &std.Manager{}
	rpt := &test.Reporter{}

	_ = mgr.AddCheck("test", health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		return &health.CheckResult{Name: "test", Status: health.StatusHealthy}
	}))
	_ = mgr.AddReporter("test", rpt)

	ctx, cancel := context.WithCancel(context.Background())
	_ = mgr.Run(ctx)

	err := mgr.AddCheck("test2", health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		return &health.CheckResult{Name: "test2", Status: health.StatusHealthy}
	}))
	if err == nil || !strings.Contains(err.Error(), "cannot add a health check to a running") {
		t.Fatalf("expected error adding check while running, got: %v", err)
	}

	cancel()
	_ = mgr.Stop(ctx)
}

// waitFor polls the condition at 10ms intervals until it returns true or the timeout expires.
func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}
