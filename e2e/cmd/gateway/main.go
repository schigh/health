package main

import (
	"context"
	"log"
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

	if err := mgr.AddCheck("orders-api",
		checkhttp.NewChecker("orders-api", "http://orders-svc:8182/health/ready",
			checkhttp.WithTimeout(3*time.Second),
		),
		health.WithCheckFrequency(health.CheckAtInterval, 5*time.Second, 0),
		health.WithLivenessImpact(),
		health.WithReadinessImpact(),
		health.WithGroup("services"),
		health.WithComponentType("http"),
		health.WithDependsOn("http://orders-svc:8182"),
	); err != nil {
		log.Fatalf("add check orders-api: %v", err)
	}

	if err := mgr.AddCheck("cluster-dns",
		dns.NewChecker("cluster-dns", "kubernetes.default.svc",
			dns.WithTimeout(2*time.Second),
		),
		health.WithCheckFrequency(health.CheckAtInterval, 10*time.Second, 0),
		health.WithStartupImpact(),
		health.WithGroup("infrastructure"),
		health.WithComponentType("dns"),
	); err != nil {
		log.Fatalf("add check cluster-dns: %v", err)
	}

	if err := mgr.AddReporter("http", httpserver.New(
		httpserver.WithPort(8181),
		httpserver.WithServiceName("gateway"),
		httpserver.WithServiceVersion("e2e"),
	)); err != nil {
		log.Fatalf("add reporter: %v", err)
	}

	errChan := mgr.Run(ctx)
	select {
	case err := <-errChan:
		log.Fatalf("manager error: %v", err)
	case <-ctx.Done():
		if err := mgr.Stop(ctx); err != nil {
			log.Printf("stop error: %v", err)
		}
	}
}
