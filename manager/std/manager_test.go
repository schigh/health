package std_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/schigh/health/v2"
	"github.com/schigh/health/v2/manager/std"
	"github.com/schigh/health/v2/reporter/test"
)

func TestManager(t *testing.T) {
	type addChecker struct {
		checker           health.Checker
		opts              []health.AddCheckOption
		expectErr         bool
		expectErrContains string
	}

	type addReporter struct {
		reporter          health.Reporter
		expectErr         bool
		expectErrContains string
	}

	type mgrTest struct {
		name                  string
		manager               *std.Manager
		checks                map[string]addChecker
		reporters             map[string]addReporter
		expire                time.Duration
		expectExpire          bool
		expectRunErr          bool
		expectRunErrContains  string
		expectStopErr         bool
		expectStopErrContains string
		usingTestReporter     bool
		report                test.Report
	}

	makeNH := func(name string) *health.CheckResult {
		return &health.CheckResult{Name: name, Status: health.StatusUnhealthy}
	}

	makeH := func(name string) *health.CheckResult {
		return &health.CheckResult{Name: name, Status: health.StatusHealthy}
	}

	// tests are named like so:
	// <type>_<num checkers>C_<num_reporters>R_<interval ms>I_<delay ms>D
	// Where <type> is one of (OT|INT) ... one-time or interval check

	tests := []mgrTest{
		{
			name:                 "OT_0C_0R_0I_0D",
			manager:              &std.Manager{},
			expectExpire:         false,
			expectRunErr:         true,
			expectRunErrContains: "there are no checkers",
		},
		{
			name:                 "OT_0C_0R_0I_0D_2",
			manager:              &std.Manager{},
			expire:               time.Second,
			expectExpire:         false,
			expectRunErr:         true,
			expectRunErrContains: "there are no checkers",
		},
		{
			name:    "OT_1C_0R_0I_10D",
			manager: &std.Manager{},
			checks: map[string]addChecker{
				"test": {
					checker: health.CheckerFunc(func(_ context.Context) *health.CheckResult {
						return &health.CheckResult{}
					}),
					opts: []health.AddCheckOption{
						health.WithCheckFrequency(health.CheckOnce, 0, 10*time.Millisecond),
					},
				},
			},
			expectExpire:         false,
			expectRunErr:         true,
			expectRunErrContains: "there are no reporters",
		},
		{
			name:    "OT_1C_1R_0I_0D",
			manager: &std.Manager{},
			checks: map[string]addChecker{
				"test": {
					checker: health.CheckerFunc(func(_ context.Context) *health.CheckResult {
						return &health.CheckResult{
							Name:   "test",
							Status: health.StatusHealthy,
						}
					}),
				},
			},
			reporters: map[string]addReporter{
				"test": {
					reporter: &test.Reporter{},
				},
			},
			expectExpire:      true,
			expire:            time.Second,
			usingTestReporter: true,
			report: test.Report{
				NumRunningStateChanges:   2,
				NumLivenessStateChanges:  1,
				NumReadinessStateChanges: 2,
				NumHealthCheckUpdates:    1,
				IsLive:                   true,
			},
		},
		{
			name:    "INT_1C_1R_100I_10D",
			manager: &std.Manager{},
			checks: map[string]addChecker{
				"test": {
					checker: health.CheckerFunc(func(_ context.Context) *health.CheckResult {
						return &health.CheckResult{Name: "test", Status: health.StatusHealthy}
					}),
					opts: []health.AddCheckOption{
						health.WithCheckFrequency(health.CheckAtInterval, 100*time.Millisecond, 10*time.Millisecond),
					},
				},
			},
			reporters: map[string]addReporter{
				"test": {
					reporter: &test.Reporter{},
				},
			},
			expectExpire:      true,
			expire:            2 * time.Second,
			usingTestReporter: true,
			report: test.Report{
				NumRunningStateChanges:   2,
				NumLivenessStateChanges:  1,
				NumReadinessStateChanges: 2,
				NumHealthCheckUpdates:    19,
				IsLive:                   true,
			},
		},
		{
			name:    "INT_1C_1R_100I_0D",
			manager: &std.Manager{},
			checks: map[string]addChecker{
				"test": {
					checker: health.CheckerFunc(func(_ context.Context) *health.CheckResult {
						return &health.CheckResult{Name: "test", Status: health.StatusHealthy}
					}),
					opts: []health.AddCheckOption{
						health.WithCheckFrequency(health.CheckAtInterval, 100*time.Millisecond, 0),
					},
				},
			},
			reporters: map[string]addReporter{
				"test": {
					reporter: &test.Reporter{},
				},
			},
			expectExpire:      true,
			expire:            2050 * time.Millisecond,
			usingTestReporter: true,
			report: test.Report{
				NumRunningStateChanges:   2,
				NumLivenessStateChanges:  1,
				NumReadinessStateChanges: 2,
				NumHealthCheckUpdates:    20,
				IsLive:                   true,
			},
		},
		{
			name:    "OT_2C_1R_0I_0D",
			manager: &std.Manager{},
			checks: map[string]addChecker{
				"test": {
					checker: health.CheckerFunc(func(_ context.Context) *health.CheckResult {
						return &health.CheckResult{Name: "test", Status: health.StatusHealthy}
					}),
				},
				"test2": {
					checker: health.CheckerFunc(func(_ context.Context) *health.CheckResult {
						return &health.CheckResult{Name: "test2", Status: health.StatusHealthy}
					}),
				},
			},
			reporters: map[string]addReporter{
				"test": {
					reporter: &test.Reporter{},
				},
			},
			expectExpire:      true,
			expire:            500 * time.Millisecond,
			usingTestReporter: true,
			report: test.Report{
				NumRunningStateChanges:   2,
				NumLivenessStateChanges:  1,
				NumReadinessStateChanges: 2,
				NumHealthCheckUpdates:    2,
				IsLive:                   true,
			},
		},
		{
			name:    "OT_2C_1R_550I_0D",
			manager: &std.Manager{},
			checks: map[string]addChecker{
				"test": {
					checker: NewMockChecker(
						makeNH("test"),
						makeNH("test"),
						makeH("test"),
					),
					opts: []health.AddCheckOption{
						health.WithCheckFrequency(health.CheckAtInterval, 550*time.Millisecond, 0),
					},
				},
			},
			reporters: map[string]addReporter{
				"test": {
					reporter: &test.Reporter{},
				},
			},
			expectExpire:      true,
			expire:            2 * time.Second,
			usingTestReporter: true,
			report: test.Report{
				NumRunningStateChanges:   2,
				NumLivenessStateChanges:  1,
				NumReadinessStateChanges: 2,
				NumHealthCheckUpdates:    2,
				IsLive:                   true,
			},
		},
		{
			name:    "OT_2C_1R_550I_0D_readiness_affected",
			manager: &std.Manager{},
			checks: map[string]addChecker{
				"test": {
					checker: NewMockChecker(
						makeNH("test"),
						makeNH("test"),
						makeH("test"),
					),
					opts: []health.AddCheckOption{
						health.WithCheckFrequency(health.CheckAtInterval, 550*time.Millisecond, 0),
						health.WithCheckImpact(false, true),
					},
				},
			},
			reporters: map[string]addReporter{
				"test": {
					reporter: &test.Reporter{},
				},
			},
			expectExpire:      true,
			expire:            2 * time.Second,
			usingTestReporter: true,
			report: test.Report{
				NumRunningStateChanges:   2,
				NumLivenessStateChanges:  1,
				NumReadinessStateChanges: 2,
				NumHealthCheckUpdates:    2,
				IsLive:                   true,
			},
		},
		{
			name:    "OT_2C_1R_550I_0D_liveness_affected",
			manager: &std.Manager{},
			checks: map[string]addChecker{
				"test": {
					checker: NewMockChecker(
						makeNH("test"),
						makeNH("test"),
						makeH("test"),
					),
					opts: []health.AddCheckOption{
						health.WithCheckFrequency(health.CheckAtInterval, 550*time.Millisecond, 0),
						health.WithCheckImpact(true, true),
					},
				},
			},
			reporters: map[string]addReporter{
				"test": {
					reporter: &test.Reporter{},
				},
			},
			expectExpire:      true,
			expire:            2 * time.Second,
			usingTestReporter: true,
			report: test.Report{
				NumRunningStateChanges:   2,
				NumLivenessStateChanges:  3,
				NumReadinessStateChanges: 2,
				NumHealthCheckUpdates:    2,
				IsLive:                   true,
			},
		},
		{
			name:    "OT_2C_1R_550I_0D_never_live_or_ready",
			manager: &std.Manager{},
			checks: map[string]addChecker{
				"test": {
					checker: NewMockChecker(
						makeNH("test"),
						makeNH("test"),
						makeNH("test"),
					),
					opts: []health.AddCheckOption{
						health.WithCheckFrequency(health.CheckAtInterval, 550*time.Millisecond, 0),
						health.WithCheckImpact(true, true),
					},
				},
			},
			reporters: map[string]addReporter{
				"test": {
					reporter: &test.Reporter{},
				},
			},
			expectExpire:      true,
			expire:            2 * time.Second,
			usingTestReporter: true,
			report: test.Report{
				NumRunningStateChanges:   2,
				NumLivenessStateChanges:  2,
				NumReadinessStateChanges: 0,
				NumHealthCheckUpdates:    3,
			},
		},
		{
			name:    "OT_2C_1R_550I_0D_flipflop",
			manager: &std.Manager{},
			checks: map[string]addChecker{
				"test": {
					checker: NewMockChecker(
						makeNH("test"),
						makeH("test"),
						makeNH("test"),
						makeH("test"),
					),
					opts: []health.AddCheckOption{
						health.WithCheckFrequency(health.CheckAtInterval, 550*time.Millisecond, 0),
						health.WithCheckImpact(true, true),
					},
				},
			},
			reporters: map[string]addReporter{
				"test": {
					reporter: &test.Reporter{},
				},
			},
			expectExpire:      true,
			expire:            2500 * time.Millisecond,
			usingTestReporter: true,
			report: test.Report{
				NumRunningStateChanges:   2,
				NumLivenessStateChanges:  5,
				NumReadinessStateChanges: 4,
				NumHealthCheckUpdates:    4,
				IsLive:                   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mgr := tt.manager

			{
				for k := range tt.checks {
					check := tt.checks[k]
					addCheckErr := mgr.AddCheck(k, check.checker, check.opts...)
					if addCheckErr != nil {
						if check.expectErr {
							if !strings.Contains(addCheckErr.Error(), check.expectErrContains) {
								t.Fatalf("expected add check error '%s' to contain '%s'", addCheckErr.Error(), check.expectErrContains)
							}
							return
						}
						t.Fatalf("unexpected error adding check: %v", addCheckErr)
					}
				}
			}

			{
				for k := range tt.reporters {
					rpt := tt.reporters[k]
					addReporterErr := mgr.AddReporter(k, rpt.reporter)
					if addReporterErr != nil {
						if rpt.expectErr {
							if !strings.Contains(addReporterErr.Error(), rpt.expectErrContains) {
								t.Fatalf("expected add reporter error '%s' to contain '%s'", addReporterErr.Error(), rpt.expectErrContains)
							}
							return
						}
					}
				}
			}

			if tt.expire > 0 {
				t.Logf("test will time out after %s", tt.expire.String())
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.expire)
				defer cancel()
			}

			var timedOut bool
			ch := mgr.Run(ctx)
			var err error
			select {
			case <-ctx.Done():
				timedOut = true
			case err = <-ch:
			}

			if err != nil {
				if !tt.expectRunErr {
					t.Fatalf("unexpected error: %v", err)
				}
				if !strings.Contains(err.Error(), tt.expectRunErrContains) {
					t.Fatalf("expected string '%s' in error: '%s'", tt.expectRunErrContains, err.Error())
				}
				return
			}

			if tt.expectExpire && !timedOut {
				t.Fatalf("the health manager was supposed to time out after %s", tt.expire.String())
			}
			if !tt.expectExpire && timedOut {
				t.Fatal("the health manager was not supposed to time out")
			}

			// stop the manager
			stopErr := mgr.Stop(ctx)
			if stopErr != nil {
				if !tt.expectStopErr {
					t.Fatalf("unexpected error: %v", stopErr)
				}
				if !strings.Contains(stopErr.Error(), tt.expectStopErrContains) {
					t.Fatalf("expected string '%s' in error: '%s'", tt.expectStopErrContains, stopErr.Error())
				}
				return
			}

			if !tt.usingTestReporter {
				return
			}

			// from here on, it is assumed that the test is using exactly one
			// instance of testReporter

			if len(tt.reporters) != 1 {
				t.Fatalf("test expected exactly one test reporter, got %d", len(tt.reporters))
			}

			rptA, ok := tt.reporters["test"]
			if !ok {
				t.Fatal("no test reporter found at index 'test'")
			}

			if rptA.reporter == nil {
				t.Fatal("no reporter set for addReporter type")
			}

			reporter, ok := test.FromReporter(rptA.reporter)
			if !ok {
				t.Fatalf("test reporter is not an instance of *test.Reporter, found %T", rptA)
			}

			// wait just a moment to allow processes to catch up
			time.Sleep(5 * time.Millisecond)

			report := reporter.Report()

			if report.IsLive != tt.report.IsLive {
				t.Fatalf("expected liveness to be reported as %t, got %t", tt.report.IsLive, report.IsLive)
			}

			if report.IsReady != tt.report.IsReady {
				t.Fatalf("expected readiness to be reported as %t, got %t", tt.report.IsReady, report.IsReady)
			}

			if report.NumLivenessStateChanges != tt.report.NumLivenessStateChanges {
				t.Fatalf("expected liveness to be updated %d times - it updated %d times", tt.report.NumLivenessStateChanges, report.NumLivenessStateChanges)
			}

			if report.NumReadinessStateChanges != tt.report.NumReadinessStateChanges {
				t.Fatalf("expected readiness to be updated %d times - it updated %d times", tt.report.NumReadinessStateChanges, report.NumReadinessStateChanges)
			}

			// the timing and other factors will often cause an off by one error if the durations are small enough
			if report.NumHealthCheckUpdates != tt.report.NumHealthCheckUpdates {
				if tt.report.NumHealthCheckUpdates > 0 {
					min := tt.report.NumHealthCheckUpdates - 1
					max := tt.report.NumHealthCheckUpdates + 1

					if report.NumHealthCheckUpdates < min || report.NumHealthCheckUpdates > max {
						t.Fatalf("expected health checks to be updated %d times - it updated %d times", tt.report.NumHealthCheckUpdates, report.NumHealthCheckUpdates)
					}
				} else {
					t.Fatalf("expected health checks to be updated %d times - it updated %d times", tt.report.NumHealthCheckUpdates, report.NumHealthCheckUpdates)
				}
			}
		})
	}
}

func TestManager_StartupProbe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	mgr := &std.Manager{}
	reporter := &test.Reporter{}

	// add a startup check that initially fails, then succeeds
	callCount := 0
	_ = mgr.AddCheck("startup_check",
		health.CheckerFunc(func(_ context.Context) *health.CheckResult {
			callCount++
			if callCount <= 2 {
				return &health.CheckResult{Name: "startup_check", Status: health.StatusUnhealthy}
			}
			return &health.CheckResult{Name: "startup_check", Status: health.StatusHealthy}
		}),
		health.WithCheckFrequency(health.CheckAtInterval, 200*time.Millisecond, 0),
		health.WithCheckImpact(true, true),
		health.WithStartupImpact(true),
	)

	_ = mgr.AddReporter("test", reporter)
	_ = mgr.Run(ctx)

	// wait for startup to complete (3 checks at 200ms each = ~600ms)
	time.Sleep(800 * time.Millisecond)

	report := reporter.Report()

	// startup should have been set to true
	if !report.IsStartup {
		t.Fatal("expected startup to be true after startup check passed")
	}

	// liveness should be true (check passed on 3rd try)
	if !report.IsLive {
		t.Fatal("expected liveness to be true after startup completed")
	}

	_ = mgr.Stop(ctx)
}

func TestManager_DegradedCheck(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mgr := &std.Manager{}
	reporter := &test.Reporter{}

	_ = mgr.AddCheck("degraded_check",
		health.CheckerFunc(func(_ context.Context) *health.CheckResult {
			return &health.CheckResult{Name: "degraded_check", Status: health.StatusDegraded}
		}),
		health.WithCheckImpact(true, true),
	)

	_ = mgr.AddReporter("test", reporter)
	_ = mgr.Run(ctx)

	time.Sleep(200 * time.Millisecond)

	report := reporter.Report()

	// degraded should NOT fail liveness or readiness
	if !report.IsLive {
		t.Fatal("expected liveness to be true for degraded check")
	}
	if !report.IsReady {
		t.Fatal("expected readiness to be true for degraded check")
	}

	_ = mgr.Stop(ctx)
}
