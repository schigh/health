package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/schigh/health/v2"
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

	mgr.AddReporter("http", httpserver.New(
		httpserver.WithPort(8183),
		httpserver.WithServiceName("payments"),
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
