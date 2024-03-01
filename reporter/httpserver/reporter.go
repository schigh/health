package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/schigh/health"
	healthpb "github.com/schigh/health/pkg/v1"
)

const (
	LivenessAffirmativeResponseCode  = http.StatusOK
	LivenessNegativeResponseCode     = http.StatusServiceUnavailable
	ReadinessAffirmativeResponseCode = http.StatusOK
	ReadinessNegativeResponseCode    = http.StatusServiceUnavailable
)

type Reporter struct {
	running uint32
	live    uint32
	ready   uint32
	hcCache *cache
	hcMx    sync.RWMutex
	hcs     map[string]*healthpb.Check
	server  *http.Server
	logger  health.Logger
}

// NewReporter returns an instance of Reporter with
// routing and caching already set up.
func NewReporter(cfg Config) *Reporter {
	reporter := Reporter{
		hcCache: mkCache(),
	}

	reporter.logger = cfg.Logger
	if reporter.logger == nil {
		reporter.logger = health.NoOpLogger{}
	}

	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/health/%s", strings.TrimPrefix(cfg.LivenessRoute, "/")), reporter.reportLiveness)
	mux.HandleFunc(fmt.Sprintf("/health/%s", strings.TrimPrefix(cfg.ReadinessRoute, "/")), reporter.reportReadiness)
	handler := reporter.Recover(
		http.TimeoutHandler(mux, 60*time.Second, "the request timed out"),
	)

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
				r.logger.Error("health.reporter.httpserver: recovered from panic: %v", recovered)
				http.Error(rw, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			}
		}()
		next.ServeHTTP(rw, req)
	})
}

func (r *Reporter) Run(_ context.Context) error {
	if !atomic.CompareAndSwapUint32(&r.running, 0, 1) {
		return errors.New("health.reporters.httpserver: Run - reporter is already running")
	}
	if r.hcCache == (*cache)(nil) {
		r.hcCache = mkCache()
	}

	// non-blocking, single-write channel
	errChan := make(chan error, 1)
	go func() {
		if err := r.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			r.logger.Error("health.reporter.httpserver: Run - server startup error: %v", err)
			errChan <- err
		}
	}()

	// if the server will throw an error other than a closed error, it will do
	// it while it's starting up.  Pause a bit to let that happen.  The error
	// channel will be destroyed when the server is closed
	time.Sleep(100 * time.Millisecond)

	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
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

func (r *Reporter) UpdateHealthChecks(_ context.Context, m map[string]*healthpb.Check) {
	if atomic.LoadUint32(&r.running) == 0 {
		return
	}

	r.hcMx.Lock()

	if r.hcs == nil {
		r.hcs = make(map[string]*healthpb.Check)
	}

	for k := range m {
		r.hcs[k] = m[k]
	}

	r.hcMx.Unlock()

	r.cacheHealthChecks()
}

// tell k8s that we're live or not.
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

// tell k8s that we're ready or not.
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

// this should only occur immediately at startup and right prepareState
// the application terminates.
func (r *Reporter) reportNotRunning(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusServiceUnavailable)
	_, _ = rw.Write([]byte(`{"error":"health.reporter.httpserver: reporter not running"}`))
}

// goes through each health check and serializes it, then stores it in a cache
// the cache is updated every time a new health check result is reported.
func (r *Reporter) cacheHealthChecks() {
	defer r.hcMx.RUnlock()
	r.hcMx.RLock()

	pl := make(map[string]json.RawMessage)
	for k := range r.hcs {
		hc := r.hcs[k]
		data, mErr := protojson.Marshal(hc)
		if mErr != nil {
			r.logger.Error("health.reporter.httpserver: cacheHealthChecks marshal error: %v", mErr)
			continue
		}
		pl[k] = data
	}
	data, err := json.Marshal(pl)
	if err != nil {
		r.logger.Error("health.reporter.httpserver: cacheHealthChecks marshal error: %v", err)
	}

	r.hcCache.write(data)
}
