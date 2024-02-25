package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/schigh/health"
	"github.com/schigh/health/manager/std"
	healthpb "github.com/schigh/health/pkg/v1"
	"github.com/schigh/health/reporter/stdout"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// create an instance of a health check manager
	mgr := std.Manager{}

	// add a spurious health check that runs at a 5-second interval
	_ = mgr.AddCheck(
		"test_check",
		health.CheckerFunc(func(ctx context.Context) *healthpb.Check {
			log.Print("Running health check")
			return &healthpb.Check{
				Name:             "test_check",
				Healthy:          true,
				AffectsLiveness:  true,
				AffectsReadiness: true,
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
