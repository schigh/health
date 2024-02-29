package httpserver_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	healthpb "github.com/schigh/health/pkg/v1"
	"github.com/schigh/health/reporter/httpserver"
)

type test struct {
	name               string
	client             *http.Client
	reporter           *httpserver.Reporter
	prepareState       func(context.Context, *testing.T, *httpserver.Reporter)
	pauseBeforeMeasure time.Duration
	measureState       func(context.Context, *testing.T, *http.Client)
}

func TestReporter(t *testing.T) {
	// single reporter instance for this test suite
	reporter := httpserver.NewReporter(httpserver.Config{
		Addr:           "0.0.0.0",
		Port:           8181,
		LivenessRoute:  "/live",
		ReadinessRoute: "/ready",
	})

	// context to close test
	ctx, cancel := context.WithCancel(context.Background())

	// stop reporter measureState all tests
	t.Cleanup(func() {
		defer cancel()
		if reporter == nil {
			return
		}
		if err := reporter.Stop(ctx); err != nil {
			t.Fatalf("reporter failed to close properly: %v", err)
		}
	})

	// start server
	if srvErr := reporter.Run(ctx); srvErr != nil {
		t.Fatalf("reporter failed to run: %v", srvErr)
	}

	// http client
	client := http.Client{
		Timeout: time.Second,
	}

	tests := mkTests(reporter, &client)

	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	// These tests are stateful for this reporter
	// DO NOT RUN IN PARALLEL
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepareState != nil {
				tt.prepareState(ctx, t, tt.reporter)
			}
			if tt.pauseBeforeMeasure > 0 {
				time.Sleep(tt.pauseBeforeMeasure)
			}
			if tt.measureState != nil {
				tt.measureState(ctx, t, tt.client)
			}
		})
	}
}

func mkTests(reporter *httpserver.Reporter, client *http.Client) []test {
	return []test{
		{
			name:     "uninitialized",
			reporter: reporter,
			client:   client,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				{
					sc, hcMap := fetchHealth(t, client, "live")
					expectStatus(t, sc, http.StatusServiceUnavailable)
					evalHealthCheckMap(t, hcMap, func(m map[string]*healthpb.Check) error {
						if len(m) != 0 {
							return fmt.Errorf("expected no health checks reported, found %d", len(m))
						}
						return nil
					})
				}

				{
					sc, hcMap := fetchHealth(t, client, "ready")
					expectStatus(t, sc, http.StatusServiceUnavailable)
					evalHealthCheckMap(t, hcMap, func(m map[string]*healthpb.Check) error {
						if len(m) != 0 {
							return fmt.Errorf("expected no health checks reported, found %d", len(m))
						}
						return nil
					})
				}
			},
		},
		{
			name:     "sets liveness to true",
			reporter: reporter,
			client:   client,
			prepareState: func(ctx context.Context, _ *testing.T, reporter *httpserver.Reporter) {
				reporter.SetLiveness(ctx, true)
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				{
					sc, hcMap := fetchHealth(t, client, "live")
					expectStatus(t, sc, http.StatusOK)
					evalHealthCheckMap(t, hcMap, func(m map[string]*healthpb.Check) error {
						if len(m) != 0 {
							return fmt.Errorf("expected no health checks reported, found %d", len(m))
						}
						return nil
					})
				}

				{
					sc, hcMap := fetchHealth(t, client, "ready")
					expectStatus(t, sc, http.StatusServiceUnavailable)
					evalHealthCheckMap(t, hcMap, func(m map[string]*healthpb.Check) error {
						if len(m) != 0 {
							return fmt.Errorf("expected no health checks reported, found %d", len(m))
						}
						return nil
					})
				}
			},
		},
		{
			name:     "sets readiness to true",
			reporter: reporter,
			client:   client,
			prepareState: func(ctx context.Context, _ *testing.T, reporter *httpserver.Reporter) {
				reporter.SetReadiness(ctx, true)
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				{
					sc, hcMap := fetchHealth(t, client, "live")
					expectStatus(t, sc, http.StatusOK)
					evalHealthCheckMap(t, hcMap, func(m map[string]*healthpb.Check) error {
						if len(m) != 0 {
							return fmt.Errorf("expected no health checks reported, found %d", len(m))
						}
						return nil
					})
				}

				{
					sc, hcMap := fetchHealth(t, client, "ready")
					expectStatus(t, sc, http.StatusOK)
					evalHealthCheckMap(t, hcMap, func(m map[string]*healthpb.Check) error {
						if len(m) != 0 {
							return fmt.Errorf("expected no health checks reported, found %d", len(m))
						}
						return nil
					})
				}
			},
		},
		{
			name:     "updates a health check",
			reporter: reporter,
			client:   client,
			prepareState: func(ctx context.Context, _ *testing.T, reporter *httpserver.Reporter) {
				reporter.UpdateHealthChecks(ctx, map[string]*healthpb.Check{
					"first": {
						Name:    "first",
						Healthy: true,
					},
				})
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				{
					sc, hcMap := fetchHealth(t, client, "live")
					expectStatus(t, sc, http.StatusOK)
					evalHealthCheckMap(t, hcMap, func(m map[string]*healthpb.Check) error {
						if len(m) != 1 {
							return fmt.Errorf("expected 1 health check reported, found %d", len(m))
						}
						return nil
					})
				}
			},
		},
		{
			name:     "updates same health check",
			reporter: reporter,
			client:   client,
			prepareState: func(ctx context.Context, _ *testing.T, reporter *httpserver.Reporter) {
				reporter.UpdateHealthChecks(ctx, map[string]*healthpb.Check{
					"first": {
						Name:    "first",
						Healthy: true,
					},
				})
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				{
					sc, hcMap := fetchHealth(t, client, "live")
					expectStatus(t, sc, http.StatusOK)
					evalHealthCheckMap(t, hcMap, func(m map[string]*healthpb.Check) error {
						if len(m) != 1 {
							return fmt.Errorf("expected 1 health check reported, found %d", len(m))
						}
						return nil
					})
				}
			},
		},
		{
			name:     "updates second health check",
			reporter: reporter,
			client:   client,
			prepareState: func(ctx context.Context, _ *testing.T, reporter *httpserver.Reporter) {
				reporter.UpdateHealthChecks(ctx, map[string]*healthpb.Check{
					"second": {
						Name:    "second",
						Healthy: true,
					},
				})
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				{
					sc, hcMap := fetchHealth(t, client, "live")
					expectStatus(t, sc, http.StatusOK)
					evalHealthCheckMap(t, hcMap, func(m map[string]*healthpb.Check) error {
						if len(m) != 2 {
							return fmt.Errorf("expected 2 health checks reported, found %d", len(m))
						}
						return nil
					})
				}
			},
		},
		{
			name:     "updates third health check",
			reporter: reporter,
			client:   client,
			prepareState: func(ctx context.Context, _ *testing.T, reporter *httpserver.Reporter) {
				reporter.UpdateHealthChecks(ctx, map[string]*healthpb.Check{
					"third": {
						Name:    "third",
						Healthy: false,
					},
				})
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				{
					sc, hcMap := fetchHealth(t, client, "live")
					expectStatus(t, sc, http.StatusOK)
					evalHealthCheckMap(t, hcMap, func(m map[string]*healthpb.Check) error {
						if len(m) != 3 {
							return fmt.Errorf("expected 3 health checks reported, found %d", len(m))
						}
						{
							hc, ok := m["third"]
							if !ok {
								return fmt.Errorf("expected health check with name 'third' not found")
							}
							if hc.GetHealthy() {
								return fmt.Errorf("expected health check named 'third' to not be healthy")
							}
						}
						{
							hc, ok := m["second"]
							if !ok {
								return fmt.Errorf("expected health check with name 'second' not found")
							}
							if !hc.GetHealthy() {
								return fmt.Errorf("expected health check named 'second' to be healthy")
							}
						}
						return nil
					})
				}
			},
		},
		{
			name:     "set readiness to false",
			reporter: reporter,
			client:   client,
			prepareState: func(ctx context.Context, _ *testing.T, reporter *httpserver.Reporter) {
				reporter.SetReadiness(ctx, false)
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				{
					sc, hcMap := fetchHealth(t, client, "live")
					expectStatus(t, sc, http.StatusOK)
					evalHealthCheckMap(t, hcMap, func(m map[string]*healthpb.Check) error {
						if len(m) != 3 {
							return fmt.Errorf("expected 3 health checks reported, found %d", len(m))
						}
						{
							hc, ok := m["third"]
							if !ok {
								return fmt.Errorf("expected health check with name 'third' not found")
							}
							if hc.GetHealthy() {
								return fmt.Errorf("expected health check named 'third' to not be healthy")
							}
						}
						{
							hc, ok := m["second"]
							if !ok {
								return fmt.Errorf("expected health check with name 'second' not found")
							}
							if !hc.GetHealthy() {
								return fmt.Errorf("expected health check named 'second' to be healthy")
							}
						}
						return nil
					})
				}
				{
					sc, _ := fetchHealth(t, client, "ready")
					expectStatus(t, sc, http.StatusServiceUnavailable)
				}
			},
		},
		{
			name:     "set readiness to true",
			reporter: reporter,
			client:   client,
			prepareState: func(ctx context.Context, _ *testing.T, reporter *httpserver.Reporter) {
				reporter.SetReadiness(ctx, true)
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				sc, _ := fetchHealth(t, client, "ready")
				expectStatus(t, sc, http.StatusOK)
			},
		},
		{
			name:     "set liveness to false",
			reporter: reporter,
			client:   client,
			prepareState: func(ctx context.Context, _ *testing.T, reporter *httpserver.Reporter) {
				reporter.SetLiveness(ctx, false)
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				{
					sc, _ := fetchHealth(t, client, "live")
					expectStatus(t, sc, http.StatusServiceUnavailable)
				}
				{
					// setting liveness to false does not affect readiness in the
					// *reporter*. The *manager* determines if readiness is
					// affected, so this should be unchanged
					sc, _ := fetchHealth(t, client, "ready")
					expectStatus(t, sc, http.StatusOK)
				}
			},
		},
		{
			name:     "set readiness to false",
			reporter: reporter,
			client:   client,
			prepareState: func(ctx context.Context, _ *testing.T, reporter *httpserver.Reporter) {
				reporter.SetReadiness(ctx, false)
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				{
					sc, _ := fetchHealth(t, client, "live")
					expectStatus(t, sc, http.StatusServiceUnavailable)
				}
				{
					sc, _ := fetchHealth(t, client, "ready")
					expectStatus(t, sc, http.StatusServiceUnavailable)
				}
			},
		},
		{
			name:     "set liveness and readiness back to true",
			reporter: reporter,
			client:   client,
			prepareState: func(ctx context.Context, _ *testing.T, reporter *httpserver.Reporter) {
				reporter.SetReadiness(ctx, true)
				reporter.SetLiveness(ctx, true)
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				{
					sc, _ := fetchHealth(t, client, "live")
					expectStatus(t, sc, http.StatusOK)
				}
				{
					sc, _ := fetchHealth(t, client, "ready")
					expectStatus(t, sc, http.StatusOK)
				}
			},
		},
	}
}

func fetchHealth(t *testing.T, client *http.Client, pathSuffix string) (int, map[string]*healthpb.Check) {
	uri := url.URL{
		Scheme: "http",
		Host:   "0.0.0.0:8181",
		Path:   path.Join("health", pathSuffix),
	}

	req, reqErr := http.NewRequestWithContext(context.Background(), http.MethodGet, uri.String(), http.NoBody)
	if reqErr != nil {
		t.Fatalf("unexpected error creating request to '%s' : %v", pathSuffix, reqErr)
	}

	resp, respErr := client.Do(req)
	if respErr != nil {
		t.Fatalf("unexpected error running request to '%s': %v", pathSuffix, respErr)
	}
	defer resp.Body.Close()

	var rawJSON map[string]json.RawMessage
	if dErr := json.NewDecoder(resp.Body).Decode(&rawJSON); dErr != nil && !errors.Is(dErr, io.EOF) {
		t.Fatalf("unexpected error decoding response from '%s': %v", pathSuffix, dErr)
	}

	out := make(map[string]*healthpb.Check)
	for k := range rawJSON {
		raw := rawJSON[k]
		var hc healthpb.Check
		if pErr := protojson.Unmarshal(raw, &hc); pErr != nil {
			t.Fatalf("unexpected error unmarshalling health check: %v", pErr)
		}
		out[k] = &hc
	}

	return resp.StatusCode, out
}

func expectStatus(t *testing.T, actual, expected int) {
	if expected != actual {
		t.Fatalf("expected status code %d, got %d instead", expected, actual)
	}
}

func evalHealthCheckMap(t *testing.T, m map[string]*healthpb.Check, f func(map[string]*healthpb.Check) error) {
	if err := f(m); err != nil {
		t.Fatalf("health check eval failed: %v", err)
	}
}
