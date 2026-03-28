package main

import (
	"context"
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

	mgr.AddCheck("postgres",
		tcp.NewChecker("postgres", "postgres:5432",
			tcp.WithTimeout(2*time.Second),
		),
		health.WithCheckFrequency(health.CheckAtInterval, 5*time.Second, 0),
		health.WithLivenessImpact(),
		health.WithReadinessImpact(),
		health.WithStartupImpact(),
		health.WithGroup("database"),
		health.WithComponentType("datastore"),
	)

	mgr.AddCheck("redis",
		redis.NewChecker("redis", "redis:6379",
			redis.WithTimeout(2*time.Second),
		),
		health.WithCheckFrequency(health.CheckAtInterval, 5*time.Second, 0),
		health.WithReadinessImpact(),
		health.WithGroup("cache"),
		health.WithComponentType("datastore"),
	)

	mgr.AddCheck("payments-api",
		checkhttp.NewChecker("payments-api", "http://payments-svc:8183/health/ready",
			checkhttp.WithTimeout(3*time.Second),
		),
		health.WithCheckFrequency(health.CheckAtInterval, 5*time.Second, 0),
		health.WithReadinessImpact(),
		health.WithGroup("services"),
		health.WithComponentType("http"),
		health.WithDependsOn("http://payments-svc:8183"),
	)

	mgr.AddReporter("http", httpserver.New(
		httpserver.WithPort(8182),
		httpserver.WithServiceName("orders"),
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
