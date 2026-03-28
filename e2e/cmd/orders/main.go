package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/schigh/health/v2"
	checkhttp "github.com/schigh/health/v2/checker/http"
	"github.com/schigh/health/v2/checker/redis"
	"github.com/schigh/health/v2/checker/tcp"
	"github.com/schigh/health/v2/manager/std"
	"github.com/schigh/health/v2/reporter/httpserver"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	mgr := std.Manager{Logger: health.DefaultLogger()}

	if err := mgr.AddCheck("postgres",
		tcp.NewChecker("postgres", "postgres:5432",
			tcp.WithTimeout(2*time.Second),
		),
		health.WithCheckFrequency(health.CheckAtInterval, 5*time.Second, 0),
		health.WithLivenessImpact(),
		health.WithReadinessImpact(),
		health.WithStartupImpact(),
		health.WithGroup("database"),
		health.WithComponentType("datastore"),
	); err != nil {
		log.Fatalf("add check postgres: %v", err)
	}

	if err := mgr.AddCheck("redis",
		redis.NewChecker("redis", "redis:6379",
			redis.WithTimeout(2*time.Second),
		),
		health.WithCheckFrequency(health.CheckAtInterval, 5*time.Second, 0),
		health.WithReadinessImpact(),
		health.WithGroup("cache"),
		health.WithComponentType("datastore"),
	); err != nil {
		log.Fatalf("add check redis: %v", err)
	}

	if err := mgr.AddCheck("payments-api",
		checkhttp.NewChecker("payments-api", "http://payments-svc:8183/readyz",
			checkhttp.WithTimeout(3*time.Second),
		),
		health.WithCheckFrequency(health.CheckAtInterval, 5*time.Second, 0),
		health.WithReadinessImpact(),
		health.WithGroup("services"),
		health.WithComponentType("http"),
		health.WithDependsOn("http://payments-svc:8183"),
	); err != nil {
		log.Fatalf("add check payments-api: %v", err)
	}

	if err := mgr.AddReporter("http", httpserver.New(
		httpserver.WithPort(8182),
		httpserver.WithServiceName("orders"),
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
