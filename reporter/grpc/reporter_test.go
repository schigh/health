package grpc_test

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/schigh/health/v2"
	healthgrpc "github.com/schigh/health/v2/reporter/grpc"
)

func startTestReporter(t *testing.T) (*healthgrpc.Reporter, grpc_health_v1.HealthClient, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	srv := grpc.NewServer()
	reporter := healthgrpc.NewReporter(healthgrpc.Config{
		Server: srv,
		Addr:   ln.Addr().String(),
	})

	go srv.Serve(ln)
	reporter.Run(context.Background())

	conn, err := grpc.NewClient(ln.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	client := grpc_health_v1.NewHealthClient(conn)
	cleanup := func() {
		conn.Close()
		reporter.Stop(context.Background())
		srv.Stop()
	}
	return reporter, client, cleanup
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
