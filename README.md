# Health

This library provides a mechanism for a service to check its health and report it 
to any type of listener. The most common use case for this library is a service 
running as a kubernetes pod, but it is useful in any running Go service.

## Concepts

### Manager
The manager is responsible for managing individual health checks and subsequent 
reporters. It handles setup and teardown of checkers and reporters. It also manages 
check frequencies and whether liveness and readiness should be affected

### Checker
A checker is anything that implements the `Checker` interface. For one-off checks, 
you can use the `CheckerFunc` type to wrap them (see example below).

### Reporter
A reporter is anything that reports the status of health checks. The current reporters are:

#### `httpserver`
This reporter runs an HTTP server (default port 8181) that reports liveness at `/health/live` 
and readiness at `/health/ready`. Passing checks will always return an HTTP 200 (OK) response, 
while failing checks will always return an HTTP 503 (Service Unavailable) response.

#### `stdout`
This reporter prints to stdout

#### `test`
this reporter can be used for tests

## Basic Example
The following is a basic example of how this library can be used
```go
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
```

## Planned updates
- ~currently the http reporter used the chi muxxer. I plan to remove it and just use servemux~
- The documentation was hastily assembled, it's definitely WIP
- add more examples
- add more basic checkers