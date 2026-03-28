package prometheus

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/schigh/health/v2"
)

// Reporter implements health.Reporter and exposes Prometheus metrics.
//
// Metrics:
//   - health_check_status (gauge, labels: check, group, component_type): 0=unhealthy, 1=degraded, 2=healthy
//   - health_check_duration_milliseconds (gauge, labels: check, group, component_type): last check duration
//   - health_check_executions_total (counter, labels: check, group, component_type, status): total executions
//   - health_liveness (gauge): 0 or 1
//   - health_readiness (gauge): 0 or 1
//   - health_startup (gauge): 0 or 1
type Reporter struct {
	running uint32
	live    uint32
	ready   uint32
	startup uint32
	hcMx    sync.RWMutex
	hcs     map[string]*health.CheckResult

	registry   *prometheus.Registry
	checkGauge *prometheus.GaugeVec
	checkDur   *prometheus.GaugeVec
	checkCount *prometheus.CounterVec
	liveGauge  prometheus.Gauge
	readyGauge prometheus.Gauge
	startGauge prometheus.Gauge
	logger     health.Logger
}

// Config configures the Prometheus reporter.
type Config struct {
	// Registry is the Prometheus registry to use. If nil, a new registry is
	// created (not the global default, to avoid conflicts).
	Registry *prometheus.Registry
	// Namespace prefixes all metric names (e.g., "myapp" → "myapp_health_check_status").
	Namespace string
	Logger    health.Logger
}

// NewReporter creates a Prometheus metrics reporter.
func NewReporter(cfg Config) *Reporter {
	reg := cfg.Registry
	if reg == nil {
		reg = prometheus.NewRegistry()
	}

	ns := cfg.Namespace
	labels := []string{"check", "group", "component_type"}

	checkGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: ns, Name: "health_check_status",
		Help: "Health check status: 0=unhealthy, 1=degraded, 2=healthy",
	}, labels)

	checkDur := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: ns, Name: "health_check_duration_milliseconds",
		Help: "Health check duration in milliseconds",
	}, labels)

	checkCount := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns, Name: "health_check_executions_total",
		Help: "Total health check executions",
	}, append(labels, "status"))

	liveGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: ns, Name: "health_liveness",
		Help: "Service liveness: 0 or 1",
	})

	readyGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: ns, Name: "health_readiness",
		Help: "Service readiness: 0 or 1",
	})

	startGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: ns, Name: "health_startup",
		Help: "Service startup: 0 or 1",
	})

	reg.MustRegister(checkGauge, checkDur, checkCount, liveGauge, readyGauge, startGauge)

	r := &Reporter{
		registry:   reg,
		checkGauge: checkGauge,
		checkDur:   checkDur,
		checkCount: checkCount,
		liveGauge:  liveGauge,
		readyGauge: readyGauge,
		startGauge: startGauge,
		logger:     cfg.Logger,
	}
	if r.logger == nil {
		r.logger = health.NoOpLogger{}
	}
	return r
}

// Handler returns an http.Handler for the /metrics endpoint.
// Mount this on your HTTP server or the httpserver reporter's mux.
func (r *Reporter) Handler() http.Handler {
	return promhttp.HandlerFor(r.registry, promhttp.HandlerOpts{})
}

func (r *Reporter) Run(_ context.Context) error {
	atomic.StoreUint32(&r.running, 1)
	return nil
}

func (r *Reporter) Stop(_ context.Context) error {
	atomic.StoreUint32(&r.running, 0)
	return nil
}

func (r *Reporter) SetLiveness(_ context.Context, b bool) {
	var v uint32
	if b {
		v = 1
	}
	atomic.StoreUint32(&r.live, v)
	r.liveGauge.Set(float64(v))
}

func (r *Reporter) SetReadiness(_ context.Context, b bool) {
	var v uint32
	if b {
		v = 1
	}
	atomic.StoreUint32(&r.ready, v)
	r.readyGauge.Set(float64(v))
}

func (r *Reporter) SetStartup(_ context.Context, b bool) {
	var v uint32
	if b {
		v = 1
	}
	atomic.StoreUint32(&r.startup, v)
	r.startGauge.Set(float64(v))
}

func (r *Reporter) UpdateHealthChecks(_ context.Context, m map[string]*health.CheckResult) {
	if atomic.LoadUint32(&r.running) == 0 {
		return
	}

	r.hcMx.Lock()
	if r.hcs == nil {
		r.hcs = make(map[string]*health.CheckResult)
	}
	for k, v := range m {
		r.hcs[k] = v
	}
	r.hcMx.Unlock()

	for _, hc := range m {
		labels := prometheus.Labels{
			"check":          hc.Name,
			"group":          hc.Group,
			"component_type": hc.ComponentType,
		}

		var statusVal float64
		switch hc.Status {
		case health.StatusHealthy:
			statusVal = 2
		case health.StatusDegraded:
			statusVal = 1
		case health.StatusUnhealthy:
			statusVal = 0
		}

		r.checkGauge.With(labels).Set(statusVal)

		if hc.Duration > 0 {
			r.checkDur.With(labels).Set(float64(hc.Duration.Milliseconds()))
		}

		statusLabels := prometheus.Labels{
			"check":          hc.Name,
			"group":          hc.Group,
			"component_type": hc.ComponentType,
			"status":         hc.Status.String(),
		}
		r.checkCount.With(statusLabels).Inc()
	}
}
