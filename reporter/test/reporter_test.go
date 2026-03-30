package test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/schigh/health/v2"
)

func TestReporter_RunStop(t *testing.T) {
	r := &Reporter{}
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error from Run: %v", err)
	}
	rpt := r.Report()
	if !rpt.IsRunning {
		t.Error("expected reporter to be running")
	}
	if rpt.NumRunningStateChanges != 1 {
		t.Errorf("expected 1 running state change, got %d", rpt.NumRunningStateChanges)
	}

	if err := r.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected error from Stop: %v", err)
	}
	rpt = r.Report()
	if rpt.IsRunning {
		t.Error("expected reporter to not be running")
	}
	if rpt.NumRunningStateChanges != 2 {
		t.Errorf("expected 2 running state changes, got %d", rpt.NumRunningStateChanges)
	}
}

func TestReporter_SetLiveness(t *testing.T) {
	r := &Reporter{}
	r.SetLiveness(context.Background(), true)
	rpt := r.Report()
	if !rpt.IsLive {
		t.Error("expected live")
	}
	if rpt.NumLivenessSetTrue != 1 {
		t.Errorf("expected 1 liveness set true, got %d", rpt.NumLivenessSetTrue)
	}

	r.SetLiveness(context.Background(), false)
	rpt = r.Report()
	if rpt.IsLive {
		t.Error("expected not live")
	}
	if rpt.NumLivenessSetFalse != 1 {
		t.Errorf("expected 1 liveness set false, got %d", rpt.NumLivenessSetFalse)
	}
	if rpt.NumLivenessStateChanges != 2 {
		t.Errorf("expected 2 liveness state changes, got %d", rpt.NumLivenessStateChanges)
	}
}

func TestReporter_SetReadiness(t *testing.T) {
	r := &Reporter{}
	r.SetReadiness(context.Background(), true)
	rpt := r.Report()
	if !rpt.IsReady {
		t.Error("expected ready")
	}
	if rpt.NumReadinessSetTrue != 1 {
		t.Errorf("expected 1 readiness set true, got %d", rpt.NumReadinessSetTrue)
	}

	r.SetReadiness(context.Background(), false)
	rpt = r.Report()
	if rpt.IsReady {
		t.Error("expected not ready")
	}
	if rpt.NumReadinessSetFalse != 1 {
		t.Errorf("expected 1 readiness set false, got %d", rpt.NumReadinessSetFalse)
	}
}

func TestReporter_SetStartup(t *testing.T) {
	r := &Reporter{}
	r.SetStartup(context.Background(), true)
	rpt := r.Report()
	if !rpt.IsStartup {
		t.Error("expected startup")
	}
	if rpt.NumStartupSetTrue != 1 {
		t.Errorf("expected 1 startup set true, got %d", rpt.NumStartupSetTrue)
	}

	r.SetStartup(context.Background(), false)
	rpt = r.Report()
	if rpt.IsStartup {
		t.Error("expected not startup")
	}
	if rpt.NumStartupSetFalse != 1 {
		t.Errorf("expected 1 startup set false, got %d", rpt.NumStartupSetFalse)
	}
}

func TestReporter_UpdateHealthChecks(t *testing.T) {
	r := &Reporter{}
	r.UpdateHealthChecks(context.Background(), map[string]*health.CheckResult{
		"db": {Name: "db", Status: health.StatusHealthy},
	})
	r.UpdateHealthChecks(context.Background(), map[string]*health.CheckResult{
		"cache": {Name: "cache", Status: health.StatusUnhealthy},
	})

	rpt := r.Report()
	if rpt.NumHealthCheckUpdates != 2 {
		t.Errorf("expected 2 health check updates, got %d", rpt.NumHealthCheckUpdates)
	}
	if len(rpt.HealthChecks) != 2 {
		t.Errorf("expected 2 health checks, got %d", len(rpt.HealthChecks))
	}
	if rpt.HealthChecks["db"].Status != health.StatusHealthy {
		t.Error("expected db to be healthy")
	}
	if rpt.HealthChecks["cache"].Status != health.StatusUnhealthy {
		t.Error("expected cache to be unhealthy")
	}
}

func TestFromReporter_Valid(t *testing.T) {
	r := &Reporter{}
	got, ok := FromReporter(r)
	if !ok {
		t.Fatal("expected ok to be true")
	}
	if got != r {
		t.Error("expected same pointer")
	}
}

func TestFromReporter_Invalid(t *testing.T) {
	_, ok := FromReporter(nil)
	if ok {
		t.Error("expected ok to be false for nil")
	}
}

func TestReport_MarshalJSON(t *testing.T) {
	r := &Reporter{}
	r.Run(context.Background())
	r.SetLiveness(context.Background(), true)
	r.SetReadiness(context.Background(), true)
	r.SetStartup(context.Background(), true)
	r.UpdateHealthChecks(context.Background(), map[string]*health.CheckResult{
		"db": {Name: "db", Status: health.StatusHealthy},
	})

	rpt := r.Report()
	data, err := json.Marshal(rpt)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if parsed["isRunning"] != true {
		t.Error("expected isRunning=true in JSON")
	}
	if parsed["isLive"] != true {
		t.Error("expected isLive=true in JSON")
	}
	hcs, ok := parsed["healthChecks"].(map[string]any)
	if !ok || len(hcs) != 1 {
		t.Error("expected 1 health check in JSON")
	}
}
