package std_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/schigh/health"
	"github.com/schigh/health/manager/std"
	healthpb "github.com/schigh/health/pkg/v1"
	"github.com/schigh/health/reporter/test"
)

func TestManager(t *testing.T) { //nolint:gocognit,gocyclo // so what
	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})

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

	makeNH := func(name string) *healthpb.Check {
		return &healthpb.Check{Name: name, Healthy: false}
	}

	makeH := func(name string) *healthpb.Check {
		return &healthpb.Check{Name: name, Healthy: true}
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
					checker: health.CheckerFunc(func(_ context.Context) *healthpb.Check {
						return &healthpb.Check{}
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
					checker: health.CheckerFunc(func(_ context.Context) *healthpb.Check {
						return &healthpb.Check{
							Name:    "test",
							Healthy: true,
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
					checker: health.CheckerFunc(func(_ context.Context) *healthpb.Check {
						return &healthpb.Check{Name: "test", Healthy: true}
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
				NumHealthCheckUpdates:    19, // will run 19 checks after a 50ms delay
				IsLive:                   true,
			},
		},
		{
			name:    "INT_1C_1R_100I_0D",
			manager: &std.Manager{},
			checks: map[string]addChecker{
				"test": {
					checker: health.CheckerFunc(func(_ context.Context) *healthpb.Check {
						return &healthpb.Check{Name: "test", Healthy: true}
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
				NumHealthCheckUpdates:    20, // will run 20 checks
				IsLive:                   true,
			},
		},
		{
			name:    "OT_2C_1R_0I_0D",
			manager: &std.Manager{},
			checks: map[string]addChecker{
				"test": {
					checker: health.CheckerFunc(func(_ context.Context) *healthpb.Check {
						return &healthpb.Check{Name: "test", Healthy: true}
					}),
				},
				"test2": {
					checker: health.CheckerFunc(func(_ context.Context) *healthpb.Check {
						return &healthpb.Check{Name: "test2", Healthy: true}
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
					checker: func() health.Checker {
						c := NewMockChecker(ctrl)
						nh1 := c.EXPECT().Check(gomock.Any()).Return(makeNH("test"))
						nh2 := c.EXPECT().Check(gomock.Any()).Return(makeNH("test")).After(nh1)
						c.EXPECT().Check(gomock.Any()).Return(makeH("test")).After(nh2)
						return c
					}(),
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
					checker: func() health.Checker {
						c := NewMockChecker(ctrl)
						nh1 := c.EXPECT().Check(gomock.Any()).Return(makeNH("test"))
						nh2 := c.EXPECT().Check(gomock.Any()).Return(makeNH("test")).After(nh1)
						c.EXPECT().Check(gomock.Any()).Return(makeH("test")).After(nh2)
						return c
					}(),
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
					checker: func() health.Checker {
						c := NewMockChecker(ctrl)
						nh1 := c.EXPECT().Check(gomock.Any()).Return(makeNH("test"))
						nh2 := c.EXPECT().Check(gomock.Any()).Return(makeNH("test")).After(nh1)
						c.EXPECT().Check(gomock.Any()).Return(makeH("test")).After(nh2)
						return c
					}(),
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
				NumLivenessStateChanges:  3, // initial on, then off due to health check, then on due to health check, then off at close
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
					checker: func() health.Checker {
						c := NewMockChecker(ctrl)
						nh1 := c.EXPECT().Check(gomock.Any()).Return(makeNH("test"))
						nh2 := c.EXPECT().Check(gomock.Any()).Return(makeNH("test")).After(nh1)
						c.EXPECT().Check(gomock.Any()).Return(makeNH("test")).After(nh2)
						return c
					}(),
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
					checker: func() health.Checker {
						c := NewMockChecker(ctrl)
						nh1 := c.EXPECT().Check(gomock.Any()).Return(makeNH("test"))
						nh2 := c.EXPECT().Check(gomock.Any()).Return(makeH("test")).After(nh1)
						nh3 := c.EXPECT().Check(gomock.Any()).Return(makeNH("test")).After(nh2)
						c.EXPECT().Check(gomock.Any()).Return(makeH("test")).After(nh3)
						return c
					}(),
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// t.Parallel()
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
						t.Fatalf("unepected error adding check: %v", addCheckErr)
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

			// wait just a moment to allow processes to catch up, specifically
			// because we are closing the manager and it may take a few ticks to
			// update the pointer flags internally
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
