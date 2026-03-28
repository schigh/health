package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/schigh/health/v2"
	"github.com/schigh/health/v2/discovery"
)

const (
	LivenessAffirmativeResponseCode  = http.StatusOK
	LivenessNegativeResponseCode     = http.StatusServiceUnavailable
	ReadinessAffirmativeResponseCode = http.StatusOK
	ReadinessNegativeResponseCode    = http.StatusServiceUnavailable
	StartupAffirmativeResponseCode   = http.StatusOK
	StartupNegativeResponseCode      = http.StatusServiceUnavailable
)

type Reporter struct {
	running uint32
	live    uint32
	ready   uint32
	startup uint32
	hcCache *cache
	hcMx    sync.RWMutex
	hcs         map[string]*health.CheckResult
	server      *http.Server
	logger      health.Logger
	serviceName string
	serviceVer  string
}

// New creates an HTTP reporter with functional options.
//
//	reporter := httpserver.New(
//	    httpserver.WithPort(9090),
//	    httpserver.WithMiddleware(httpserver.BasicAuth("user", "pass")),
//	)
func New(opts ...Option) *Reporter {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}
	return newFromConfig(cfg)
}

// NewReporter creates an HTTP reporter from a Config struct.
// For functional options, use New() instead.
func NewReporter(cfg Config) *Reporter {
	return newFromConfig(cfg)
}

func newFromConfig(cfg Config) *Reporter {
	reporter := Reporter{
		hcCache:     mkCache(),
		serviceName: cfg.ServiceName,
		serviceVer:  cfg.ServiceVersion,
	}

	reporter.logger = cfg.Logger
	if reporter.logger == nil {
		reporter.logger = health.NoOpLogger{}
	}

	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/health/%s", strings.TrimPrefix(cfg.LivenessRoute, "/")), reporter.reportLiveness)
	mux.HandleFunc(fmt.Sprintf("/health/%s", strings.TrimPrefix(cfg.ReadinessRoute, "/")), reporter.reportReadiness)
	mux.HandleFunc(fmt.Sprintf("/health/%s", strings.TrimPrefix(cfg.StartupRoute, "/")), reporter.reportStartup)
	mux.HandleFunc("/.well-known/health", reporter.reportManifest)

	var handler http.Handler
	handler = http.TimeoutHandler(mux, 60*time.Second, "the request timed out")

	// apply middleware in reverse order so the first middleware
	// in the list is the outermost (first to see the request)
	for i := len(cfg.Middleware) - 1; i >= 0; i-- {
		handler = cfg.Middleware[i](handler)
	}

	handler = reporter.Recover(handler)

	reporter.server = &http.Server{
		ReadHeaderTimeout: 3 * time.Second,
		Addr:              fmt.Sprintf("%s:%d", cfg.Addr, cfg.Port),
		Handler:           handler,
	}

	return &reporter
}

func (r *Reporter) Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				r.logger.Error("recovered from panic", "error", recovered)
				http.Error(rw, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			}
		}()
		next.ServeHTTP(rw, req)
	})
}

func (r *Reporter) Run(_ context.Context) error {
	if !atomic.CompareAndSwapUint32(&r.running, 0, 1) {
		return errors.New("health.reporter.httpserver: Run - reporter is already running")
	}
	if r.hcCache == (*cache)(nil) {
		r.hcCache = mkCache()
	}

	ln, err := net.Listen("tcp", r.server.Addr)
	if err != nil {
		atomic.StoreUint32(&r.running, 0)
		return fmt.Errorf("health.reporter.httpserver: Run - listen error: %w", err)
	}

	go func() {
		if err := r.server.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
			r.logger.Error("server error", "error", err)
		}
	}()

	return nil
}

func (r *Reporter) Stop(ctx context.Context) error {
	_ = atomic.SwapUint32(&r.running, 0)
	return r.server.Shutdown(ctx)
}

func (r *Reporter) SetLiveness(_ context.Context, b bool) {
	var v uint32
	if b {
		v = 1
	}
	_ = atomic.SwapUint32(&r.live, v)
}

func (r *Reporter) SetReadiness(_ context.Context, b bool) {
	var v uint32
	if b {
		v = 1
	}
	_ = atomic.SwapUint32(&r.ready, v)
}

func (r *Reporter) SetStartup(_ context.Context, b bool) {
	var v uint32
	if b {
		v = 1
	}
	_ = atomic.SwapUint32(&r.startup, v)
}

func (r *Reporter) UpdateHealthChecks(_ context.Context, m map[string]*health.CheckResult) {
	if atomic.LoadUint32(&r.running) == 0 {
		return
	}

	r.hcMx.Lock()

	if r.hcs == nil {
		r.hcs = make(map[string]*health.CheckResult)
	}

	for k := range m {
		r.hcs[k] = m[k]
	}

	r.hcMx.Unlock()

	r.cacheHealthChecks()
}

func (r *Reporter) reportLiveness(rw http.ResponseWriter, rq *http.Request) {
	if atomic.LoadUint32(&r.running) == 0 {
		r.reportNotRunning(rw, rq)
		return
	}

	statusCode := LivenessAffirmativeResponseCode
	if atomic.LoadUint32(&r.live) == 0 {
		statusCode = LivenessNegativeResponseCode
	}

	data := r.hcCache.read()
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(statusCode)
	_, _ = rw.Write(data)
}

func (r *Reporter) reportReadiness(rw http.ResponseWriter, rq *http.Request) {
	if atomic.LoadUint32(&r.running) == 0 {
		r.reportNotRunning(rw, rq)
		return
	}

	statusCode := ReadinessAffirmativeResponseCode
	if atomic.LoadUint32(&r.ready) == 0 {
		statusCode = ReadinessNegativeResponseCode
	}

	data := r.hcCache.read()
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(statusCode)
	_, _ = rw.Write(data)
}

func (r *Reporter) reportStartup(rw http.ResponseWriter, rq *http.Request) {
	if atomic.LoadUint32(&r.running) == 0 {
		r.reportNotRunning(rw, rq)
		return
	}

	statusCode := StartupAffirmativeResponseCode
	if atomic.LoadUint32(&r.startup) == 0 {
		statusCode = StartupNegativeResponseCode
	}

	data := r.hcCache.read()
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(statusCode)
	_, _ = rw.Write(data)
}

// reportManifest serves the /.well-known/health discovery manifest.
func (r *Reporter) reportManifest(rw http.ResponseWriter, rq *http.Request) {
	if atomic.LoadUint32(&r.running) == 0 {
		r.reportNotRunning(rw, rq)
		return
	}

	r.hcMx.RLock()
	defer r.hcMx.RUnlock()

	status := "pass"
	if atomic.LoadUint32(&r.live) == 0 {
		status = "fail"
	} else if atomic.LoadUint32(&r.ready) == 0 {
		status = "warn"
	}

	checks := make([]discovery.CheckEntry, 0, len(r.hcs))
	for _, hc := range r.hcs {
		entry := discovery.CheckEntry{
			Name:             hc.Name,
			Status:           hc.Status.String(),
			Group:            hc.Group,
			ComponentType:    hc.ComponentType,
			AffectsLiveness:  hc.AffectsLiveness,
			AffectsReadiness: hc.AffectsReadiness,
			AffectsStartup:   hc.AffectsStartup,
			DependsOn:        hc.DependsOn,
		}
		if hc.Duration > 0 {
			entry.Duration = hc.Duration.String()
		}
		if !hc.Timestamp.IsZero() {
			entry.LastCheck = hc.Timestamp.Format(time.RFC3339)
		}
		if hc.Error != nil {
			entry.Error = hc.Error.Error()
		}
		checks = append(checks, entry)
	}

	manifest := discovery.Manifest{
		Service:   r.serviceName,
		Version:   r.serviceVer,
		Status:    status,
		Checks:    checks,
		Timestamp: time.Now(),
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(rw).Encode(manifest)
}

// reportNotRunning should only occur immediately at startup and right before
// the application terminates.
func (r *Reporter) reportNotRunning(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusServiceUnavailable)
	_, _ = rw.Write([]byte(`{"error":"health.reporter.httpserver: reporter not running"}`))
}

// cacheHealthChecks serializes all health checks into JSON and stores the result
// in the cache. The cache is updated every time a new health check result is reported.
func (r *Reporter) cacheHealthChecks() {
	defer func() {
		if rec := recover(); rec != nil {
			r.logger.Error("cacheHealthChecks panicked", "error", rec)
		}
	}()

	r.hcMx.RLock()
	defer r.hcMx.RUnlock()

	type checkJSON struct {
		Name             string            `json:"name"`
		Status           string            `json:"status"`
		AffectsLiveness  bool              `json:"affectsLiveness"`
		AffectsReadiness bool              `json:"affectsReadiness"`
		AffectsStartup   bool              `json:"affectsStartup,omitempty"`
		Group            string            `json:"group,omitempty"`
		ComponentType    string            `json:"componentType,omitempty"`
		Error            string            `json:"error,omitempty"`
		ErrorSince       string            `json:"errorSince,omitempty"`
		Duration         string            `json:"duration,omitempty"`
		LastCheck        string            `json:"lastCheck,omitempty"`
		Metadata         map[string]string `json:"metadata,omitempty"`
	}

	pl := make(map[string]checkJSON)
	for k, hc := range r.hcs {
		cj := checkJSON{
			Name:             hc.Name,
			Status:           hc.Status.String(),
			AffectsLiveness:  hc.AffectsLiveness,
			AffectsReadiness: hc.AffectsReadiness,
			AffectsStartup:   hc.AffectsStartup,
			Group:            hc.Group,
			ComponentType:    hc.ComponentType,
			Metadata:         hc.Metadata,
		}
		if hc.Error != nil {
			cj.Error = hc.Error.Error()
		}
		if !hc.ErrorSince.IsZero() {
			cj.ErrorSince = hc.ErrorSince.Format(time.RFC3339)
		}
		if hc.Duration > 0 {
			cj.Duration = hc.Duration.String()
		}
		if !hc.Timestamp.IsZero() {
			cj.LastCheck = hc.Timestamp.Format(time.RFC3339)
		}
		pl[k] = cj
	}

	data, err := json.Marshal(pl)
	if err != nil {
		r.logger.Error("cacheHealthChecks marshal error", "error", err)
		return
	}

	r.hcCache.write(data)
}
