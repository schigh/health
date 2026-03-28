package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/schigh/health/v2"
	checkhttp "github.com/schigh/health/v2/checker/http"
	"github.com/schigh/health/v2/checker/dns"
	"github.com/schigh/health/v2/manager/std"
	"github.com/schigh/health/v2/reporter/httpserver"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	mgr := std.Manager{Logger: health.DefaultLogger()}

	mgr.AddCheck("orders-api",
		checkhttp.NewChecker("orders-api", "http://orders-svc:8182/health/ready",
			checkhttp.WithTimeout(3*time.Second),
		),
		health.WithCheckFrequency(health.CheckAtInterval, 5*time.Second, 0),
		health.WithLivenessImpact(),
		health.WithReadinessImpact(),
		health.WithGroup("services"),
		health.WithComponentType("http"),
		health.WithDependsOn("http://orders-svc:8182"),
	)

	mgr.AddCheck("cluster-dns",
		dns.NewChecker("cluster-dns", "kubernetes.default.svc",
			dns.WithTimeout(2*time.Second),
		),
		health.WithCheckFrequency(health.CheckAtInterval, 10*time.Second, 0),
		health.WithStartupImpact(),
		health.WithGroup("infrastructure"),
		health.WithComponentType("dns"),
	)

	mgr.AddReporter("http", httpserver.New(
		httpserver.WithPort(8181),
		httpserver.WithServiceName("gateway"),
		httpserver.WithServiceVersion("e2e"),
	))

	errChan := mgr.Run(ctx)
	select {
	case err := <-errChan:
		panic(err)
	case <-ctx.Done():
		mgr.Stop(ctx)
	}
}
