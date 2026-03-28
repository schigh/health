package otel

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/schigh/health/v2"
)

// Reporter implements health.Reporter and emits OpenTelemetry metrics.
//
// Metrics emitted:
//   - health.check.status (gauge, per check): 0=unhealthy, 1=degraded, 2=healthy
//   - health.check.duration (histogram, per check): check execution time in milliseconds
//   - health.check.executions (counter, per check): total check executions by status
//   - health.liveness (gauge): 0 or 1
//   - health.readiness (gauge): 0 or 1
//   - health.startup (gauge): 0 or 1
type Reporter struct {
	running uint32
	live    uint32
	ready   uint32
	startup uint32
	hcMx    sync.RWMutex
	hcs     map[string]*health.CheckResult

	meter      metric.Meter
	checkGauge metric.Int64Gauge
	checkHist  metric.Float64Histogram
	checkCount metric.Int64Counter
	liveGauge  metric.Int64Gauge
	readyGauge metric.Int64Gauge
	startGauge metric.Int64Gauge
	logger     health.Logger
}

// Config configures the OTel reporter.
type Config struct {
	// MeterProvider supplies the meter. If nil, the global provider is NOT used
	// automatically; you must provide one.
	MeterProvider metric.MeterProvider
	Logger        health.Logger
}

// NewReporter creates an OTel metrics reporter.
func NewReporter(cfg Config) (*Reporter, error) {
	if cfg.MeterProvider == nil {
		return nil, errNoMeterProvider
	}

	meter := cfg.MeterProvider.Meter("github.com/schigh/health/v2/reporter/otel")

	checkGauge, err := meter.Int64Gauge("health.check.status",
		metric.WithDescription("Health check status: 0=unhealthy, 1=degraded, 2=healthy"),
	)
	if err != nil {
		return nil, err
	}

	checkHist, err := meter.Float64Histogram("health.check.duration",
		metric.WithDescription("Health check duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	checkCount, err := meter.Int64Counter("health.check.executions",
		metric.WithDescription("Total health check executions"),
	)
	if err != nil {
		return nil, err
	}

	liveGauge, err := meter.Int64Gauge("health.liveness",
		metric.WithDescription("Service liveness: 0 or 1"),
	)
	if err != nil {
		return nil, err
	}

	readyGauge, err := meter.Int64Gauge("health.readiness",
		metric.WithDescription("Service readiness: 0 or 1"),
	)
	if err != nil {
		return nil, err
	}

	startGauge, err := meter.Int64Gauge("health.startup",
		metric.WithDescription("Service startup: 0 or 1"),
	)
	if err != nil {
		return nil, err
	}

	r := &Reporter{
		meter:      meter,
		checkGauge: checkGauge,
		checkHist:  checkHist,
		checkCount: checkCount,
		liveGauge:  liveGauge,
		readyGauge: readyGauge,
		startGauge: startGauge,
		logger:     cfg.Logger,
	}
	if r.logger == nil {
		r.logger = health.NoOpLogger{}
	}
	return r, nil
}

func (r *Reporter) Run(_ context.Context) error {
	atomic.StoreUint32(&r.running, 1)
	return nil
}

func (r *Reporter) Stop(_ context.Context) error {
	atomic.StoreUint32(&r.running, 0)
	return nil
}

func (r *Reporter) SetLiveness(ctx context.Context, b bool) {
	var v uint32
	if b {
		v = 1
	}
	atomic.StoreUint32(&r.live, v)
	r.liveGauge.Record(ctx, int64(v))
}

func (r *Reporter) SetReadiness(ctx context.Context, b bool) {
	var v uint32
	if b {
		v = 1
	}
	atomic.StoreUint32(&r.ready, v)
	r.readyGauge.Record(ctx, int64(v))
}

func (r *Reporter) SetStartup(ctx context.Context, b bool) {
	var v uint32
	if b {
		v = 1
	}
	atomic.StoreUint32(&r.startup, v)
	r.startGauge.Record(ctx, int64(v))
}

func (r *Reporter) UpdateHealthChecks(ctx context.Context, m map[string]*health.CheckResult) {
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
		attrs := []attribute.KeyValue{
			attribute.String("check", hc.Name),
		}
		if hc.Group != "" {
			attrs = append(attrs, attribute.String("group", hc.Group))
		}
		if hc.ComponentType != "" {
			attrs = append(attrs, attribute.String("component_type", hc.ComponentType))
		}
		attrSet := metric.WithAttributes(attrs...)

		// status gauge: 0=unhealthy, 1=degraded, 2=healthy
		var statusVal int64
		switch hc.Status {
		case health.StatusHealthy:
			statusVal = 2
		case health.StatusDegraded:
			statusVal = 1
		case health.StatusUnhealthy:
			statusVal = 0
		}
		r.checkGauge.Record(ctx, statusVal, attrSet)

		// duration histogram
		if hc.Duration > 0 {
			r.checkHist.Record(ctx, float64(hc.Duration)/float64(time.Millisecond), attrSet)
		}

		// execution counter with status label
		r.checkCount.Add(ctx, 1, metric.WithAttributes(
			append(attrs, attribute.String("status", hc.Status.String()))...,
		))
	}
}

var errNoMeterProvider = errorString("health.reporter.otel: MeterProvider is required")

type errorString string

func (e errorString) Error() string { return string(e) }
