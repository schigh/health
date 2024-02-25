package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/schigh/health"
	"github.com/schigh/health/checker/db"
	"github.com/schigh/health/manager/std"
	"github.com/schigh/health/reporter/test"
)

var pinger db.CtxPinger

func TestMain(m *testing.M) {
	time.Sleep(5 * time.Second)

	dbx, err := sqlx.Connect("mysql", "root:root@tcp(localhost:3306)/testdb")
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}

	pinger = dbx
	_ = m.Run()

	if cErr := dbx.Close(); cErr != nil {
		log.Fatalf("db close error: %v", err)
	}
}

func TestChecker(t *testing.T) {
	// do not run parallel
	mgr := std.Manager{}
	reporter := test.Reporter{}
	if rErr := mgr.AddReporter("test", &reporter); rErr != nil {
		t.Fatalf("unexpected error adding test reporter: %v", rErr)
	}

	checker := db.NewChecker("db", pinger, db.WithTimeout(time.Second))
	if cErr := mgr.AddCheck("db", checker, health.WithCheckFrequency(health.CheckAtInterval, 2*time.Second, 0)); cErr != nil {
		t.Fatalf("unexpected error adding checker: %v", cErr)
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithTimeout(context.Background(), 11*time.Second)
	defer cancel()

	errChan := mgr.Run(ctx)

	select {
	case <-stopChan:
		if err := mgr.Stop(ctx); err != nil {
			t.Fatalf("unexpected error stopping manager: %v", err)
		}
	case <-ctx.Done():
		if err := mgr.Stop(ctx); err != nil {
			t.Fatalf("unexpected error stopping manager: %v", err)
		}
	case err := <-errChan:
		t.Fatalf("unexpected error running manager: %v", err)
	}

	rpt := reporter.Report()
	data, mErr := json.MarshalIndent(rpt, "", "  ")
	if mErr != nil {
		t.Fatalf("unexpected error marshaling report: %v", mErr)
	}

	fmt.Println(string(data))

	// - The test ran for 11 seconds
	// - The checker ran once every 2 seconds
	// - There should be 5 checks
	if rpt.NumHealthCheckUpdates != 5 {
		t.Errorf("expected 5 health check updates, got %d", rpt.NumHealthCheckUpdates)
	}

	// - The manager started with liveness and readiness set false
	// - The manager became live right away and ready after the first check
	// - The manager stopped being live or ready when the manager stopped
	// - The manager was set live=true once, and ready=true once
	// - The manager was set ready=false once
	if rpt.NumLivenessStateChanges != 1 {
		t.Errorf("expected liveness to change 2 times, got %d", rpt.NumLivenessStateChanges)
	}
	if rpt.NumReadinessStateChanges != 2 {
		t.Errorf("expected readiness to change 2 times, got %d", rpt.NumReadinessStateChanges)
	}
	if rpt.NumLivenessSetTrue != 1 {
		t.Errorf("expected liveness to be set true 1 time, got %d", rpt.NumLivenessSetTrue)
	}
	if rpt.NumReadinessSetTrue != 1 {
		t.Errorf("expected readiness to be set true 1 time, got %d", rpt.NumReadinessSetTrue)
	}
	if rpt.NumLivenessSetFalse != 0 {
		t.Errorf("expected liveness to be set false 1 time, got %d", rpt.NumLivenessSetFalse)
	}
	if rpt.NumReadinessSetFalse != 1 {
		t.Errorf("expected readiness to be set false 1 time, got %d", rpt.NumReadinessSetFalse)
	}
}

func TestCheckerTimeout(t *testing.T) {
	// do not run parallel
	mgr := std.Manager{}
	reporter := test.Reporter{}
	if rErr := mgr.AddReporter("test", &reporter); rErr != nil {
		t.Fatalf("unexpected error adding test reporter: %v", rErr)
	}

	// this be a short interval
	checker := db.NewChecker("db", pinger, db.WithTimeout(time.Microsecond))
	if cErr := mgr.AddCheck("db", checker, health.WithCheckFrequency(health.CheckAtInterval, 100*time.Millisecond, 0)); cErr != nil {
		t.Fatalf("unexpected error adding checker: %v", cErr)
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	errChan := mgr.Run(ctx)

	select {
	case <-stopChan:
		if err := mgr.Stop(ctx); err != nil {
			t.Fatalf("unexpected error stopping manager: %v", err)
		}
	case <-ctx.Done():
		if err := mgr.Stop(ctx); err != nil {
			t.Fatalf("unexpected error stopping manager: %v", err)
		}
	case err := <-errChan:
		t.Fatalf("unexpected error running manager: %v", err)
	}

	rpt := reporter.Report()
	data, mErr := json.MarshalIndent(rpt, "", "  ")
	if mErr != nil {
		t.Fatalf("unexpected error marshaling report: %v", mErr)
	}

	fmt.Println(string(data))

	// - The test ran for 1 second
	// - The checker ran once every 100ms
	// - There should be 9 checks
	if rpt.NumHealthCheckUpdates != 9 {
		t.Errorf("expected 9 health check updates, got %d", rpt.NumHealthCheckUpdates)
	}

	// - The manager started with liveness and readiness set false
	// - The manager became live right away and ready after the first check
	// - The manager stopped being live or ready when the manager stopped
	// - The manager was set live=true once, and ready=true once
	// - The manager was set ready=false once
	if rpt.NumLivenessStateChanges != 1 {
		t.Errorf("expected liveness to change 2 times, got %d", rpt.NumLivenessStateChanges)
	}
	if rpt.NumReadinessStateChanges != 2 {
		t.Errorf("expected readiness to change 2 times, got %d", rpt.NumReadinessStateChanges)
	}
	if rpt.NumLivenessSetTrue != 1 {
		t.Errorf("expected liveness to be set true 1 time, got %d", rpt.NumLivenessSetTrue)
	}
	if rpt.NumReadinessSetTrue != 1 {
		t.Errorf("expected readiness to be set true 1 time, got %d", rpt.NumReadinessSetTrue)
	}
	if rpt.NumLivenessSetFalse != 0 {
		t.Errorf("expected liveness to be set false 1 time, got %d", rpt.NumLivenessSetFalse)
	}
	if rpt.NumReadinessSetFalse != 1 {
		t.Errorf("expected readiness to be set false 1 time, got %d", rpt.NumReadinessSetFalse)
	}
}
