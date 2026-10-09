package grpc_test

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	healthgrpc "github.com/schigh/health/reporter/grpc/v2"
	"github.com/schigh/health/v2"
)

func startTestReporter(t *testing.T) (*healthgrpc.Reporter, grpc_health_v1.HealthClient, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	srv := grpc.NewServer()
	reporter := healthgrpc.NewReporter(healthgrpc.Config{Server: srv})

	go srv.Serve(ln)
	if err := reporter.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	client := newTestClient(t, ln.Addr().String())
	cleanup := func() {
		reporter.Stop(context.Background())
		srv.Stop()
	}
	return reporter, client, cleanup
}

func newTestClient(t *testing.T, addr string) grpc_health_v1.HealthClient {
	t.Helper()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return grpc_health_v1.NewHealthClient(conn)
}

// freeAddr returns a local address that was free when checked.
func freeAddr(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}

func TestReporter_CallerServer_StopLeavesServerServing(t *testing.T) {
	reporter, client, cleanup := startTestReporter(t)
	defer cleanup()

	if err := reporter.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	if _, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{}); err != nil {
		t.Fatalf("expected the caller's server to keep serving after Stop, got %v", err)
	}
}

func TestReporter_OwnServer_ServesOnAddrAndStops(t *testing.T) {
	addr := freeAddr(t)
	reporter := healthgrpc.NewReporter(healthgrpc.Config{Addr: addr})
	if err := reporter.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	client := newTestClient(t, addr)

	if _, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{}); err != nil {
		t.Fatalf("expected the reporter's server to serve on %s, got %v", addr, err)
	}

	if err := reporter.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{}); err == nil {
		t.Fatal("expected Check to fail after Stop stopped the reporter's server")
	}
}

func TestReporter_OwnServer_RequiresAddr(t *testing.T) {
	reporter := healthgrpc.NewReporter(healthgrpc.Config{})
	if err := reporter.Run(context.Background()); err == nil {
		t.Fatal("expected Run to fail with no Server and no Addr")
	}
}

func TestReporter_OverallHealth_NotServing(t *testing.T) {
	_, client, cleanup := startTestReporter(t)
	defer cleanup()

	resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("expected NOT_SERVING, got %s", resp.Status)
	}
}

func TestReporter_OverallHealth_Serving(t *testing.T) {
	reporter, client, cleanup := startTestReporter(t)
	defer cleanup()

	reporter.SetLiveness(context.Background(), true)
	reporter.SetReadiness(context.Background(), true)

	resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("expected SERVING, got %s", resp.Status)
	}
}

func TestReporter_NamedCheck_Unknown(t *testing.T) {
	_, client, cleanup := startTestReporter(t)
	defer cleanup()

	resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{Service: "nonexistent"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN {
		t.Fatalf("expected SERVICE_UNKNOWN, got %s", resp.Status)
	}
}

func TestReporter_NamedCheck_Healthy(t *testing.T) {
	reporter, client, cleanup := startTestReporter(t)
	defer cleanup()

	reporter.UpdateHealthChecks(context.Background(), map[string]*health.CheckResult{
		"postgres": {Name: "postgres", Status: health.StatusHealthy},
	})

	resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{Service: "postgres"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("expected SERVING for healthy check, got %s", resp.Status)
	}
}

func TestReporter_NamedCheck_Unhealthy(t *testing.T) {
	reporter, client, cleanup := startTestReporter(t)
	defer cleanup()

	reporter.UpdateHealthChecks(context.Background(), map[string]*health.CheckResult{
		"redis": {Name: "redis", Status: health.StatusUnhealthy},
	})

	resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{Service: "redis"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("expected NOT_SERVING for unhealthy check, got %s", resp.Status)
	}
}

func TestReporter_Watch(t *testing.T) {
	reporter, client, cleanup := startTestReporter(t)
	defer cleanup()

	reporter.SetLiveness(context.Background(), true)
	reporter.SetReadiness(context.Background(), true)

	stream, err := client.Watch(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("expected SERVING from Watch, got %s", resp.Status)
	}
}
