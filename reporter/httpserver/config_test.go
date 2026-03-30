package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestBasicAuth_ValidCredentials(t *testing.T) {
	handler := BasicAuth("admin", "secret")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	req.SetBasicAuth("admin", "secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestReportNotRunning(t *testing.T) {
	r := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18481,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})
	// don't call Run — reporter stays in not-running state

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.reportNotRunning(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "reporter not running") {
		t.Errorf("expected 'reporter not running' in body, got %q", body)
	}
}

func TestReportManifest(t *testing.T) {
	r := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18482,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
		ServiceName:    "test-svc",
		ServiceVersion: "1.0.0",
	})
	r.running = 1
	r.live = 1
	r.ready = 1

	r.hcs = map[string]*health.CheckResult{
		"db": {
			Name:             "db",
			Status:           health.StatusHealthy,
			Group:            "database",
			ComponentType:    "datastore",
			AffectsLiveness:  true,
			AffectsReadiness: true,
			Duration:         100 * time.Millisecond,
			Timestamp:        time.Now(),
		},
		"cache": {
			Name:   "cache",
			Status: health.StatusUnhealthy,
			Error:  errors.New("connection refused"),
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/health", nil)
	r.reportManifest(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var manifest map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &manifest); err != nil {
		t.Fatalf("failed to decode manifest: %v", err)
	}
	if manifest["service"] != "test-svc" {
		t.Errorf("expected service 'test-svc', got %v", manifest["service"])
	}
	if manifest["status"] != "pass" {
		t.Errorf("expected status 'pass', got %v", manifest["status"])
	}
}

func TestReportManifest_StatusVariants(t *testing.T) {
	r := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18483,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})
	r.running = 1
	r.hcs = make(map[string]*health.CheckResult)

	// live=0 → fail
	r.live = 0
	r.ready = 1
	rr := httptest.NewRecorder()
	r.reportManifest(rr, httptest.NewRequest(http.MethodGet, "/.well-known/health", nil))
	var m map[string]any
	json.Unmarshal(rr.Body.Bytes(), &m)
	if m["status"] != "fail" {
		t.Errorf("expected 'fail' when not live, got %v", m["status"])
	}

	// live=1, ready=0 → warn
	r.live = 1
	r.ready = 0
	rr = httptest.NewRecorder()
	r.reportManifest(rr, httptest.NewRequest(http.MethodGet, "/.well-known/health", nil))
	json.Unmarshal(rr.Body.Bytes(), &m)
	if m["status"] != "warn" {
		t.Errorf("expected 'warn' when live but not ready, got %v", m["status"])
	}
}

func TestReportManifest_NotRunning(t *testing.T) {
	r := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18484,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})
	// running=0

	rr := httptest.NewRecorder()
	r.reportManifest(rr, httptest.NewRequest(http.MethodGet, "/.well-known/health", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when not running, got %d", rr.Code)
	}
}

func TestRecover_Panic(t *testing.T) {
	r := &Reporter{logger: health.NoOpLogger{}}
	handler := r.Recover(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 after panic, got %d", rr.Code)
	}
}

func TestRunAlreadyRunning(t *testing.T) {
	r := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18485,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})
	ctx := context.Background()
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
	defer r.Stop(ctx)

	err := r.Run(ctx)
	if err == nil || !strings.Contains(err.Error(), "already running") {
		t.Errorf("expected 'already running' error, got %v", err)
	}
}

func TestCacheHealthChecks_WithAllFields(t *testing.T) {
	r := &Reporter{
		running: 1,
		hcCache: mkCache(),
		logger:  health.NoOpLogger{},
		hcs: map[string]*health.CheckResult{
			"db": {
				Name:             "db",
				Status:           health.StatusUnhealthy,
				AffectsLiveness:  true,
				AffectsReadiness: true,
				AffectsStartup:   true,
				Group:            "database",
				ComponentType:    "datastore",
				Error:            errors.New("connection refused"),
				ErrorSince:       time.Now().Add(-time.Minute),
				Duration:         50 * time.Millisecond,
				Timestamp:        time.Now(),
				Metadata:         map[string]string{"host": "db.local"},
			},
		},
	}

	r.cacheHealthChecks()
	data := r.hcCache.read()

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse cached JSON: %v", err)
	}

	db, ok := parsed["db"].(map[string]any)
	if !ok {
		t.Fatal("expected 'db' entry in cached JSON")
	}
	if db["status"] != "unhealthy" {
		t.Errorf("expected 'unhealthy', got %v", db["status"])
	}
	if db["error"] != "connection refused" {
		t.Errorf("expected error message, got %v", db["error"])
	}
	if db["group"] != "database" {
		t.Errorf("expected group 'database', got %v", db["group"])
	}
	if db["duration"] == nil || db["duration"] == "" {
		t.Error("expected duration to be set")
	}
	if db["errorSince"] == nil || db["errorSince"] == "" {
		t.Error("expected errorSince to be set")
	}
}

func TestReportLiveness_NotRunning(t *testing.T) {
	r := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18486,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})
	// running=0
	rr := httptest.NewRecorder()
	r.reportLiveness(rr, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestReportReadiness_NotRunning(t *testing.T) {
	r := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18487,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})
	rr := httptest.NewRecorder()
	r.reportReadiness(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestReportStartup_NotRunning(t *testing.T) {
	r := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18488,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})
	rr := httptest.NewRecorder()
	r.reportStartup(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestReportReadiness_Verbose(t *testing.T) {
	r := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18489,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})
	r.running = 1
	r.ready = 1
	r.hcs = map[string]*health.CheckResult{
		"db": {Name: "db", Status: health.StatusHealthy},
	}

	rr := httptest.NewRecorder()
	r.reportReadiness(rr, httptest.NewRequest(http.MethodGet, "/readyz?verbose", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "[+]db ok") {
		t.Errorf("expected verbose output, got %q", body)
	}
}

func TestReportStartup_Verbose(t *testing.T) {
	r := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18490,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})
	r.running = 1
	r.startup = 1
	r.hcs = map[string]*health.CheckResult{
		"db": {Name: "db", Status: health.StatusHealthy},
	}

	rr := httptest.NewRecorder()
	r.reportStartup(rr, httptest.NewRequest(http.MethodGet, "/healthz?verbose", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestReportIndividualCheck_NotRunning(t *testing.T) {
	r := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18491,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})
	// running=0
	rr := httptest.NewRecorder()
	r.reportIndividualCheck(rr, httptest.NewRequest(http.MethodGet, "/livez/db", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestReportIndividualCheck_EmptyName(t *testing.T) {
	r := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18492,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})
	r.running = 1
	r.hcs = make(map[string]*health.CheckResult)

	// path doesn't match any prefix + "/"
	rr := httptest.NewRecorder()
	r.reportIndividualCheck(rr, httptest.NewRequest(http.MethodGet, "/unknown/db", nil))
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unmatched prefix, got %d", rr.Code)
	}
}

func TestReportIndividualCheck_UnhealthyWithoutError(t *testing.T) {
	r := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18493,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})
	r.running = 1
	r.hcs = map[string]*health.CheckResult{
		"db": {Name: "db", Status: health.StatusUnhealthy},
	}

	rr := httptest.NewRecorder()
	r.reportIndividualCheck(rr, httptest.NewRequest(http.MethodGet, "/livez/db", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "check failed") {
		t.Errorf("expected 'check failed' default message, got %q", body)
	}
}

func TestUpdateHealthChecks_NotRunning(t *testing.T) {
	r := &Reporter{running: 0, hcCache: mkCache(), logger: health.NoOpLogger{}}
	r.UpdateHealthChecks(context.Background(), map[string]*health.CheckResult{
		"db": {Name: "db", Status: health.StatusHealthy},
	})
	// should be a no-op, hcs map should remain nil
	if r.hcs != nil {
		t.Error("expected hcs to be nil when not running")
	}
}

func TestReportVerbose_UnhealthyWithoutError(t *testing.T) {
	r := &Reporter{
		running: 1,
		hcCache: mkCache(),
		logger:  health.NoOpLogger{},
		hcs: map[string]*health.CheckResult{
			"db": {Name: "db", Status: health.StatusUnhealthy},
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/livez?verbose", nil)
	r.reportVerbose(rr, req)

	body := rr.Body.String()
	if !strings.Contains(body, "check failed") {
		t.Errorf("expected default 'check failed' for nil error, got %q", body)
	}
}

func TestRunListenError(t *testing.T) {
	// Start a reporter on a port
	r1 := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18494,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})
	ctx := context.Background()
	if err := r1.Run(ctx); err != nil {
		t.Fatal(err)
	}
	defer r1.Stop(ctx)

	// Try to start a second reporter on the same port
	r2 := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18494,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})
	err := r2.Run(ctx)
	if err == nil || !strings.Contains(err.Error(), "listen error") {
		t.Errorf("expected listen error, got %v", err)
	}
}

func TestReportIndividualCheck_ReadyzPath(t *testing.T) {
	r := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18495,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})
	r.running = 1
	r.hcs = map[string]*health.CheckResult{
		"db": {Name: "db", Status: health.StatusHealthy},
	}

	rr := httptest.NewRecorder()
	r.reportIndividualCheck(rr, httptest.NewRequest(http.MethodGet, "/readyz/db", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "[+]db ok") {
		t.Errorf("expected '[+]db ok', got %q", body)
	}
}

func TestReportIndividualCheck_HealthzPath(t *testing.T) {
	r := newFromConfig(Config{
		Addr:           "127.0.0.1",
		Port:           18496,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})
	r.running = 1
	r.hcs = map[string]*health.CheckResult{
		"db": {Name: "db", Status: health.StatusHealthy},
	}

	rr := httptest.NewRecorder()
	r.reportIndividualCheck(rr, httptest.NewRequest(http.MethodGet, "/healthz/db", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

// suppress unused import warnings
var _ = io.Discard
var _ = errors.New
var _ = time.Second

func TestBasicAuth_InvalidCredentials(t *testing.T) {
	handler := BasicAuth("admin", "secret")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// wrong password
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	req.SetBasicAuth("admin", "wrong")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("wrong password: expected 401, got %d", rr.Code)
	}
	if rr.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected WWW-Authenticate header")
	}

	// wrong username
	req = httptest.NewRequest(http.MethodGet, "/livez", nil)
	req.SetBasicAuth("wrong", "secret")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("wrong username: expected 401, got %d", rr.Code)
	}

	// no auth header
	req = httptest.NewRequest(http.MethodGet, "/livez", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("no auth: expected 401, got %d", rr.Code)
	}
}
