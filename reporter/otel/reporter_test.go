package otel_test

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/schigh/health/v2"
	healthotel "github.com/schigh/health/v2/reporter/otel"
)

func setupReporter(t *testing.T) (*healthotel.Reporter, *metric.ManualReader) {
	t.Helper()

	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))

	r, err := healthotel.NewReporter(healthotel.Config{
		MeterProvider: provider,
	})
	if err != nil {
		t.Fatal(err)
	}
	r.Run(context.Background())
	return r, reader
}

func collectMetrics(t *testing.T, reader *metric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	return rm
}

func findMetric(rm metricdata.ResourceMetrics, name string) *metricdata.Metrics {
	for _, sm := range rm.ScopeMetrics {
		for i := range sm.Metrics {
			if sm.Metrics[i].Name == name {
				return &sm.Metrics[i]
			}
		}
	}
	return nil
}

func TestReporter_NilMeterProvider(t *testing.T) {
	_, err := healthotel.NewReporter(healthotel.Config{})
	if err == nil {
		t.Fatal("expected error for nil MeterProvider")
	}
}

func TestReporter_LivenessMetric(t *testing.T) {
	r, reader := setupReporter(t)

	r.SetLiveness(context.Background(), true)
	rm := collectMetrics(t, reader)

	m := findMetric(rm, "health.liveness")
	if m == nil {
		t.Fatal("health.liveness metric not found")
	}
}

func TestReporter_ReadinessMetric(t *testing.T) {
	r, reader := setupReporter(t)

	r.SetReadiness(context.Background(), true)
	rm := collectMetrics(t, reader)

	m := findMetric(rm, "health.readiness")
	if m == nil {
		t.Fatal("health.readiness metric not found")
	}
}

func TestReporter_CheckStatusMetric(t *testing.T) {
	r, reader := setupReporter(t)

	r.UpdateHealthChecks(context.Background(), map[string]*health.CheckResult{
		"postgres": {
			Name:          "postgres",
			Status:        health.StatusHealthy,
			Group:         "database",
			ComponentType: "datastore",
			Duration:      2 * time.Millisecond,
		},
	})

	rm := collectMetrics(t, reader)

	status := findMetric(rm, "health.check.status")
	if status == nil {
		t.Fatal("health.check.status metric not found")
	}

	dur := findMetric(rm, "health.check.duration")
	if dur == nil {
		t.Fatal("health.check.duration metric not found")
	}

	count := findMetric(rm, "health.check.executions")
	if count == nil {
		t.Fatal("health.check.executions metric not found")
	}
}

func TestReporter_NotRunning(t *testing.T) {
	r, reader := setupReporter(t)
	r.Stop(context.Background())

	r.UpdateHealthChecks(context.Background(), map[string]*health.CheckResult{
		"test": {Name: "test", Status: health.StatusHealthy},
	})

	rm := collectMetrics(t, reader)
	// check.status should not exist since reporter was stopped
	status := findMetric(rm, "health.check.status")
	if status != nil {
		t.Fatal("expected no check metrics when reporter is stopped")
	}
}
