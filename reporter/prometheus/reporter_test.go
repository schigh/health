package prometheus_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/schigh/health/v2"
	healthprom "github.com/schigh/health/v2/reporter/prometheus"
)

func TestReporter_MetricsEndpoint(t *testing.T) {
	r := healthprom.NewReporter(healthprom.Config{})
	r.Run(context.Background())

	r.SetLiveness(context.Background(), true)
	r.SetReadiness(context.Background(), true)
	r.UpdateHealthChecks(context.Background(), map[string]*health.CheckResult{
		"postgres": {
			Name:          "postgres",
			Status:        health.StatusHealthy,
			Group:         "database",
			ComponentType: "datastore",
			Duration:      5 * time.Millisecond,
		},
	})

	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	text := string(body)

	// verify expected metrics exist
	for _, want := range []string{
		"health_liveness 1",
		"health_readiness 1",
		"health_check_status",
		"health_check_duration_milliseconds",
		"health_check_executions_total",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("expected metrics to contain %q, got:\n%s", want, text)
		}
	}
}

func TestReporter_Namespace(t *testing.T) {
	r := healthprom.NewReporter(healthprom.Config{Namespace: "myapp"})
	r.Run(context.Background())
	r.SetLiveness(context.Background(), true)

	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "myapp_health_liveness 1") {
		t.Errorf("expected namespaced metric, got:\n%s", body)
	}
}

func TestReporter_UnhealthyCheck(t *testing.T) {
	r := healthprom.NewReporter(healthprom.Config{})
	r.Run(context.Background())
	r.UpdateHealthChecks(context.Background(), map[string]*health.CheckResult{
		"redis": {Name: "redis", Status: health.StatusUnhealthy},
	})

	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `health_check_status{check="redis"`) {
		t.Errorf("expected redis check metric, got:\n%s", body)
	}
}

func TestReporter_NotRunning(t *testing.T) {
	r := healthprom.NewReporter(healthprom.Config{})
	// don't call Run — reporter not running
	r.UpdateHealthChecks(context.Background(), map[string]*health.CheckResult{
		"test": {Name: "test", Status: health.StatusHealthy},
	})

	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// should not contain check metrics since reporter wasn't running
	if strings.Contains(string(body), "health_check_status") {
		t.Errorf("expected no check metrics when not running, got:\n%s", body)
	}
}
