package grpc

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/schigh/health/v2"
)

// Reporter implements health.Reporter and serves the standard gRPC
// health checking protocol (grpc.health.v1.Health).
type Reporter struct {
	running uint32
	live    uint32
	ready   uint32
	startup uint32
	hcMx    sync.RWMutex
	hcs     map[string]*health.CheckResult
	server  *grpc.Server
	addr    string
	logger  health.Logger
}

// Config configures the gRPC health reporter.
type Config struct {
	// Server is an existing gRPC server to register on. If nil, a new
	// server is created and managed by this reporter.
	Server *grpc.Server
	// Addr is the listen address (e.g., "0.0.0.0:8182"). Required if
	// Server is nil.
	Addr   string
	Logger health.Logger
}

// NewReporter creates a gRPC health reporter.
func NewReporter(cfg Config) *Reporter {
	r := &Reporter{
		addr:   cfg.Addr,
		logger: cfg.Logger,
	}
	if r.logger == nil {
		r.logger = health.NoOpLogger{}
	}
	if cfg.Server != nil {
		r.server = cfg.Server
	} else {
		r.server = grpc.NewServer()
	}
	grpc_health_v1.RegisterHealthServer(r.server, r)
	return r
}

func (r *Reporter) Run(_ context.Context) error {
	if !atomic.CompareAndSwapUint32(&r.running, 0, 1) {
		return fmt.Errorf("health.reporter.grpc: already running")
	}

	ln, err := net.Listen("tcp", r.addr)
	if err != nil {
		atomic.StoreUint32(&r.running, 0)
		return fmt.Errorf("health.reporter.grpc: listen error: %w", err)
	}

	go func() {
		if err := r.server.Serve(ln); err != nil {
			r.logger.Error("gRPC server error", "error", err)
		}
	}()

	return nil
}

func (r *Reporter) Stop(_ context.Context) error {
	if !atomic.CompareAndSwapUint32(&r.running, 1, 0) {
		return nil
	}
	r.server.GracefulStop()
	return nil
}

func (r *Reporter) SetLiveness(_ context.Context, b bool) {
	var v uint32
	if b {
		v = 1
	}
	atomic.StoreUint32(&r.live, v)
}

func (r *Reporter) SetReadiness(_ context.Context, b bool) {
	var v uint32
	if b {
		v = 1
	}
	atomic.StoreUint32(&r.ready, v)
}

func (r *Reporter) SetStartup(_ context.Context, b bool) {
	var v uint32
	if b {
		v = 1
	}
	atomic.StoreUint32(&r.startup, v)
}

func (r *Reporter) UpdateHealthChecks(_ context.Context, m map[string]*health.CheckResult) {
	r.hcMx.Lock()
	defer r.hcMx.Unlock()

	if r.hcs == nil {
		r.hcs = make(map[string]*health.CheckResult)
	}
	for k, v := range m {
		r.hcs[k] = v
	}
}

// Check implements grpc_health_v1.HealthServer.
func (r *Reporter) Check(_ context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	svc := req.GetService()

	// empty service name = overall health
	if svc == "" {
		status := grpc_health_v1.HealthCheckResponse_SERVING
		if atomic.LoadUint32(&r.live) == 0 || atomic.LoadUint32(&r.ready) == 0 {
			status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
		}
		return &grpc_health_v1.HealthCheckResponse{Status: status}, nil
	}

	// named service = individual check
	r.hcMx.RLock()
	hc, ok := r.hcs[svc]
	r.hcMx.RUnlock()

	if !ok {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN,
		}, nil
	}

	status := grpc_health_v1.HealthCheckResponse_SERVING
	if hc.Status == health.StatusUnhealthy {
		status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}

	return &grpc_health_v1.HealthCheckResponse{Status: status}, nil
}

// List implements grpc_health_v1.HealthServer and returns the status of all
// known services.
func (r *Reporter) List(_ context.Context, _ *grpc_health_v1.HealthListRequest) (*grpc_health_v1.HealthListResponse, error) {
	r.hcMx.RLock()
	defer r.hcMx.RUnlock()

	statuses := make(map[string]*grpc_health_v1.HealthCheckResponse, len(r.hcs))
	for name, hc := range r.hcs {
		status := grpc_health_v1.HealthCheckResponse_SERVING
		if hc.Status == health.StatusUnhealthy {
			status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
		}
		statuses[name] = &grpc_health_v1.HealthCheckResponse{
			Status: status,
		}
	}

	return &grpc_health_v1.HealthListResponse{Statuses: statuses}, nil
}

// Watch implements grpc_health_v1.HealthServer (streaming).
// This is a minimal implementation that sends the current status and returns.
func (r *Reporter) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	resp, err := r.Check(stream.Context(), req)
	if err != nil {
		return err
	}
	return stream.Send(resp)
}
