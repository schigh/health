package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/schigh/health/v2"
	"github.com/schigh/health/v2/manager/std"
	"github.com/schigh/health/v2/reporter/stdout"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// create an instance of a health check manager
	mgr := std.Manager{}

	// add a health check that runs at a 5-second interval
	_ = mgr.AddCheck(
		"test_check",
		health.CheckerFunc(func(_ context.Context) *health.CheckResult {
			log.Print("Running health check")
			return &health.CheckResult{
				Name:   "test_check",
				Status: health.StatusHealthy,
			}
		}),
		health.WithCheckFrequency(health.CheckAtInterval, 5*time.Second, 0),
		health.WithCheckImpact(true, true),
	)

	// add a reporter
	_ = mgr.AddReporter("stdout", &stdout.Reporter{})

	// run the manager
	// This returns a read-only error channel. You can ignore this if you like.
	errChan := mgr.Run(ctx)

	// this toy example won't pipe any errors, but the handling
	// is here to demonstrate how that should be done
	for {
		select {
		case err := <-errChan:
			log.Printf("error: %v", err)
		case <-ctx.Done():
			_ = mgr.Stop(ctx) // remember to stop the manager
			return
		}
	}
}
