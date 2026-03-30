package httpserver

import (
	"net/http"
	"testing"

	"github.com/schigh/health/v2"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.Addr != "0.0.0.0" {
		t.Errorf("expected addr '0.0.0.0', got %q", cfg.Addr)
	}
	if cfg.Port != 8181 {
		t.Errorf("expected port 8181, got %d", cfg.Port)
	}
	if cfg.LivenessRoute != "/livez" {
		t.Errorf("expected liveness route '/livez', got %q", cfg.LivenessRoute)
	}
	if cfg.ReadinessRoute != "/readyz" {
		t.Errorf("expected readiness route '/readyz', got %q", cfg.ReadinessRoute)
	}
	if cfg.StartupRoute != "/healthz" {
		t.Errorf("expected startup route '/healthz', got %q", cfg.StartupRoute)
	}
}

func TestOptions(t *testing.T) {
	logger := health.NoOpLogger{}
	mw := func(next http.Handler) http.Handler { return next }

	opts := []Option{
		WithAddr("127.0.0.1"),
		WithPort(9090),
		WithLivenessRoute("/live"),
		WithReadinessRoute("/ready"),
		WithStartupRoute("/startup"),
		WithLogger(logger),
		WithMiddleware(mw),
		WithServiceName("test-svc"),
		WithServiceVersion("1.0.0"),
	}

	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}

	if cfg.Addr != "127.0.0.1" {
		t.Errorf("expected addr '127.0.0.1', got %q", cfg.Addr)
	}
	if cfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Port)
	}
	if cfg.LivenessRoute != "/live" {
		t.Errorf("expected liveness route '/live', got %q", cfg.LivenessRoute)
	}
	if cfg.ReadinessRoute != "/ready" {
		t.Errorf("expected readiness route '/ready', got %q", cfg.ReadinessRoute)
	}
	if cfg.StartupRoute != "/startup" {
		t.Errorf("expected startup route '/startup', got %q", cfg.StartupRoute)
	}
	if cfg.ServiceName != "test-svc" {
		t.Errorf("expected service name 'test-svc', got %q", cfg.ServiceName)
	}
	if cfg.ServiceVersion != "1.0.0" {
		t.Errorf("expected service version '1.0.0', got %q", cfg.ServiceVersion)
	}
	if len(cfg.Middleware) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(cfg.Middleware))
	}
}

func TestNew(t *testing.T) {
	r := New(WithPort(18383), WithServiceName("svc"), WithServiceVersion("v1"))
	if r == nil {
		t.Fatal("expected non-nil reporter")
	}
	if r.serviceName != "svc" {
		t.Errorf("expected service name 'svc', got %q", r.serviceName)
	}
	if r.serviceVer != "v1" {
		t.Errorf("expected service version 'v1', got %q", r.serviceVer)
	}
}

func TestBasicAuth(t *testing.T) {
	mw := BasicAuth("user", "pass")
	if mw == nil {
		t.Fatal("expected non-nil middleware")
	}
}
