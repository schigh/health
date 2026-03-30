package health_test

import (
	"context"
	"testing"
	"time"

	"github.com/schigh/health/v2"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status health.Status
		want   string
	}{
		{health.StatusHealthy, "healthy"},
		{health.StatusDegraded, "degraded"},
		{health.StatusUnhealthy, "unhealthy"},
		{health.Status(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("Status(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestCheckerFunc(t *testing.T) {
	called := false
	cf := health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		called = true
		return &health.CheckResult{Name: "fn", Status: health.StatusHealthy}
	})

	result := cf.Check(context.Background())
	if !called {
		t.Fatal("expected CheckerFunc to be called")
	}
	if result.Name != "fn" {
		t.Errorf("expected name 'fn', got %q", result.Name)
	}
}

func TestWithCheckFrequency(t *testing.T) {
	var opts health.AddCheckOptions
	health.WithCheckFrequency(health.CheckAtInterval, 5*time.Second, 2*time.Second)(&opts)

	if opts.Frequency != health.CheckAtInterval {
		t.Errorf("expected CheckAtInterval, got %v", opts.Frequency)
	}
	if opts.Interval != 5*time.Second {
		t.Errorf("expected 5s interval, got %v", opts.Interval)
	}
	if opts.Delay != 2*time.Second {
		t.Errorf("expected 2s delay, got %v", opts.Delay)
	}
}

func TestWithLivenessImpact(t *testing.T) {
	var opts health.AddCheckOptions
	health.WithLivenessImpact()(&opts)
	if !opts.AffectsLiveness {
		t.Error("expected AffectsLiveness to be true")
	}
}

func TestWithReadinessImpact(t *testing.T) {
	var opts health.AddCheckOptions
	health.WithReadinessImpact()(&opts)
	if !opts.AffectsReadiness {
		t.Error("expected AffectsReadiness to be true")
	}
}

func TestWithStartupImpact(t *testing.T) {
	var opts health.AddCheckOptions
	health.WithStartupImpact()(&opts)
	if !opts.AffectsStartup {
		t.Error("expected AffectsStartup to be true")
	}
}

func TestWithGroup(t *testing.T) {
	var opts health.AddCheckOptions
	health.WithGroup("database")(&opts)
	if opts.Group != "database" {
		t.Errorf("expected group 'database', got %q", opts.Group)
	}
}

func TestWithComponentType(t *testing.T) {
	var opts health.AddCheckOptions
	health.WithComponentType("datastore")(&opts)
	if opts.ComponentType != "datastore" {
		t.Errorf("expected component type 'datastore', got %q", opts.ComponentType)
	}
}

func TestWithDependsOn(t *testing.T) {
	var opts health.AddCheckOptions
	health.WithDependsOn("http://svc-a/healthz", "http://svc-b/healthz")(&opts)
	if len(opts.DependsOn) != 2 {
		t.Fatalf("expected 2 deps, got %d", len(opts.DependsOn))
	}
	if opts.DependsOn[0] != "http://svc-a/healthz" {
		t.Errorf("expected first dep 'http://svc-a/healthz', got %q", opts.DependsOn[0])
	}
}

func TestDefaultLogger(t *testing.T) {
	l := health.DefaultLogger()
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
	// should not panic
	l.Debug("test")
	l.Info("test")
	l.Warn("test")
	l.Error("test")
}

func TestNoOpLogger(t *testing.T) {
	l := health.NoOpLogger{}
	// should not panic
	l.Debug("test")
	l.Info("test")
	l.Warn("test")
	l.Error("test")
}

func TestWithDependsOn_Additive(t *testing.T) {
	var opts health.AddCheckOptions
	health.WithDependsOn("http://a")(&opts)
	health.WithDependsOn("http://b")(&opts)
	if len(opts.DependsOn) != 2 {
		t.Fatalf("expected 2 deps after two calls, got %d", len(opts.DependsOn))
	}
}

func TestCheckerFunc_NilResult(t *testing.T) {
	cf := health.CheckerFunc(func(_ context.Context) *health.CheckResult {
		return nil
	})
	result := cf.Check(context.Background())
	if result != nil {
		t.Error("expected nil result")
	}
}
