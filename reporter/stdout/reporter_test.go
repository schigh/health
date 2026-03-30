package stdout

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/schigh/health/v2"
)

func TestReporter_RunStop(t *testing.T) {
	r := &Reporter{}
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error from Run: %v", err)
	}
	if err := r.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected error from Stop: %v", err)
	}
}

func TestReporter_SetLiveness(t *testing.T) {
	r := &Reporter{}
	r.SetLiveness(context.Background(), true)
	if r.live != 1 {
		t.Error("expected live to be 1")
	}
	r.SetLiveness(context.Background(), false)
	if r.live != 0 {
		t.Error("expected live to be 0")
	}
}

func TestReporter_SetReadiness(t *testing.T) {
	r := &Reporter{}
	r.SetReadiness(context.Background(), true)
	if r.ready != 1 {
		t.Error("expected ready to be 1")
	}
	r.SetReadiness(context.Background(), false)
	if r.ready != 0 {
		t.Error("expected ready to be 0")
	}
}

func TestReporter_SetStartup(t *testing.T) {
	r := &Reporter{}
	r.SetStartup(context.Background(), true)
	if r.startup != 1 {
		t.Error("expected startup to be 1")
	}
	r.SetStartup(context.Background(), false)
	if r.startup != 0 {
		t.Error("expected startup to be 0")
	}
}

func TestReporter_UpdateHealthChecks(t *testing.T) {
	// Capture output via a temp file since w is *os.File
	tmp, err := os.CreateTemp("", "stdout-reporter-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	origW := w
	w = tmp
	defer func() { w = origW }()

	r := &Reporter{}
	r.SetLiveness(context.Background(), true)
	r.SetReadiness(context.Background(), false)
	r.SetStartup(context.Background(), true)

	r.UpdateHealthChecks(context.Background(), map[string]*health.CheckResult{
		"db": {
			Name:             "db",
			Status:           health.StatusHealthy,
			AffectsLiveness:  true,
			AffectsReadiness: false,
		},
	})

	// Read back what was written
	tmp.Sync()
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	output := string(data)

	if !strings.Contains(output, "yes") {
		t.Error("expected 'yes' for live/startup in output")
	}
	if !strings.Contains(output, "db") {
		t.Error("expected check name 'db' in output")
	}
	if !strings.Contains(output, "healthy") {
		t.Error("expected 'healthy' status in output")
	}
}
