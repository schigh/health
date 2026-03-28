package httpserver_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/schigh/health/v2"
	"github.com/schigh/health/v2/reporter/httpserver"
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
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})

	// context to close test
	ctx, cancel := context.WithCancel(context.Background())

	// stop reporter after all tests
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

	// These tests are stateful for this reporter
	// DO NOT RUN IN PARALLEL
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
					sc, hcMap := fetchHealth(t, client, "livez")
					expectStatus(t, sc, http.StatusServiceUnavailable)
					evalHealthCheckMap(t, hcMap, func(m map[string]checkJSON) error {
						if len(m) != 0 {
							return fmt.Errorf("expected no health checks reported, found %d", len(m))
						}
						return nil
					})
				}

				{
					sc, hcMap := fetchHealth(t, client, "readyz")
					expectStatus(t, sc, http.StatusServiceUnavailable)
					evalHealthCheckMap(t, hcMap, func(m map[string]checkJSON) error {
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
					sc, hcMap := fetchHealth(t, client, "livez")
					expectStatus(t, sc, http.StatusOK)
					evalHealthCheckMap(t, hcMap, func(m map[string]checkJSON) error {
						if len(m) != 0 {
							return fmt.Errorf("expected no health checks reported, found %d", len(m))
						}
						return nil
					})
				}

				{
					sc, hcMap := fetchHealth(t, client, "readyz")
					expectStatus(t, sc, http.StatusServiceUnavailable)
					evalHealthCheckMap(t, hcMap, func(m map[string]checkJSON) error {
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
					sc, hcMap := fetchHealth(t, client, "livez")
					expectStatus(t, sc, http.StatusOK)
					evalHealthCheckMap(t, hcMap, func(m map[string]checkJSON) error {
						if len(m) != 0 {
							return fmt.Errorf("expected no health checks reported, found %d", len(m))
						}
						return nil
					})
				}

				{
					sc, hcMap := fetchHealth(t, client, "readyz")
					expectStatus(t, sc, http.StatusOK)
					evalHealthCheckMap(t, hcMap, func(m map[string]checkJSON) error {
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
				reporter.UpdateHealthChecks(ctx, map[string]*health.CheckResult{
					"first": {
						Name:   "first",
						Status: health.StatusHealthy,
					},
				})
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				{
					sc, hcMap := fetchHealth(t, client, "livez")
					expectStatus(t, sc, http.StatusOK)
					evalHealthCheckMap(t, hcMap, func(m map[string]checkJSON) error {
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
				reporter.UpdateHealthChecks(ctx, map[string]*health.CheckResult{
					"first": {
						Name:   "first",
						Status: health.StatusHealthy,
					},
				})
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				{
					sc, hcMap := fetchHealth(t, client, "livez")
					expectStatus(t, sc, http.StatusOK)
					evalHealthCheckMap(t, hcMap, func(m map[string]checkJSON) error {
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
				reporter.UpdateHealthChecks(ctx, map[string]*health.CheckResult{
					"second": {
						Name:   "second",
						Status: health.StatusHealthy,
					},
				})
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				{
					sc, hcMap := fetchHealth(t, client, "livez")
					expectStatus(t, sc, http.StatusOK)
					evalHealthCheckMap(t, hcMap, func(m map[string]checkJSON) error {
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
				reporter.UpdateHealthChecks(ctx, map[string]*health.CheckResult{
					"third": {
						Name:   "third",
						Status: health.StatusUnhealthy,
					},
				})
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				{
					sc, hcMap := fetchHealth(t, client, "livez")
					expectStatus(t, sc, http.StatusOK)
					evalHealthCheckMap(t, hcMap, func(m map[string]checkJSON) error {
						if len(m) != 3 {
							return fmt.Errorf("expected 3 health checks reported, found %d", len(m))
						}
						{
							hc, ok := m["third"]
							if !ok {
								return fmt.Errorf("expected health check with name 'third' not found")
							}
							if hc.Status != "unhealthy" {
								return fmt.Errorf("expected health check named 'third' to be unhealthy, got %s", hc.Status)
							}
						}
						{
							hc, ok := m["second"]
							if !ok {
								return fmt.Errorf("expected health check with name 'second' not found")
							}
							if hc.Status != "healthy" {
								return fmt.Errorf("expected health check named 'second' to be healthy, got %s", hc.Status)
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
					sc, _ := fetchHealth(t, client, "livez")
					expectStatus(t, sc, http.StatusOK)
				}
				{
					sc, _ := fetchHealth(t, client, "readyz")
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
				sc, _ := fetchHealth(t, client, "readyz")
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
					sc, _ := fetchHealth(t, client, "livez")
					expectStatus(t, sc, http.StatusServiceUnavailable)
				}
				{
					sc, _ := fetchHealth(t, client, "readyz")
					expectStatus(t, sc, http.StatusOK)
				}
			},
		},
		{
			name:     "set readiness to false 2",
			reporter: reporter,
			client:   client,
			prepareState: func(ctx context.Context, _ *testing.T, reporter *httpserver.Reporter) {
				reporter.SetReadiness(ctx, false)
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				{
					sc, _ := fetchHealth(t, client, "livez")
					expectStatus(t, sc, http.StatusServiceUnavailable)
				}
				{
					sc, _ := fetchHealth(t, client, "readyz")
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
					sc, _ := fetchHealth(t, client, "livez")
					expectStatus(t, sc, http.StatusOK)
				}
				{
					sc, _ := fetchHealth(t, client, "readyz")
					expectStatus(t, sc, http.StatusOK)
				}
			},
		},
		{
			name:     "startup probe defaults to not started",
			reporter: reporter,
			client:   client,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				sc, _ := fetchHealth(t, client, "healthz")
				expectStatus(t, sc, http.StatusServiceUnavailable)
			},
		},
		{
			name:     "set startup to true",
			reporter: reporter,
			client:   client,
			prepareState: func(ctx context.Context, _ *testing.T, reporter *httpserver.Reporter) {
				reporter.SetStartup(ctx, true)
			},
			pauseBeforeMeasure: 10 * time.Millisecond,
			measureState: func(_ context.Context, t *testing.T, client *http.Client) {
				sc, _ := fetchHealth(t, client, "healthz")
				expectStatus(t, sc, http.StatusOK)
			},
		},
	}
}

type checkJSON struct {
	Name             string            `json:"name"`
	Status           string            `json:"status"`
	AffectsLiveness  bool              `json:"affectsLiveness"`
	AffectsReadiness bool              `json:"affectsReadiness"`
	AffectsStartup   bool              `json:"affectsStartup,omitempty"`
	Error            string            `json:"error,omitempty"`
	ErrorSince       string            `json:"errorSince,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

func fetchHealth(t *testing.T, client *http.Client, pathSuffix string) (int, map[string]checkJSON) {
	uri := url.URL{
		Scheme: "http",
		Host:   "0.0.0.0:8181",
		Path:   "/" + pathSuffix,
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

	var out map[string]checkJSON
	if dErr := json.NewDecoder(resp.Body).Decode(&out); dErr != nil && !errors.Is(dErr, io.EOF) {
		t.Fatalf("unexpected error decoding response from '%s': %v", pathSuffix, dErr)
	}
	if out == nil {
		out = make(map[string]checkJSON)
	}

	return resp.StatusCode, out
}

func expectStatus(t *testing.T, actual, expected int) {
	if expected != actual {
		t.Fatalf("expected status code %d, got %d instead", expected, actual)
	}
}

func evalHealthCheckMap(t *testing.T, m map[string]checkJSON, f func(map[string]checkJSON) error) {
	if err := f(m); err != nil {
		t.Fatalf("health check eval failed: %v", err)
	}
}

func TestIndividualCheck(t *testing.T) {
	reporter := httpserver.NewReporter(httpserver.Config{
		Addr:           "0.0.0.0",
		Port:           8281,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		defer cancel()
		reporter.Stop(ctx)
	})

	if err := reporter.Run(ctx); err != nil {
		t.Fatal(err)
	}

	reporter.SetLiveness(ctx, true)
	reporter.SetReadiness(ctx, true)

	reporter.UpdateHealthChecks(ctx, map[string]*health.CheckResult{
		"postgres": {Name: "postgres", Status: health.StatusHealthy},
		"redis":    {Name: "redis", Status: health.StatusUnhealthy, Error: errors.New("connection refused")},
	})

	client := http.Client{Timeout: time.Second}

	// individual healthy check
	{
		uri := "http://0.0.0.0:8281/livez/postgres"
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, uri, http.NoBody)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			t.Errorf("/livez/postgres: expected 200, got %d", resp.StatusCode)
		}
		if !strings.Contains(string(body), "[+]postgres ok") {
			t.Errorf("expected '[+]postgres ok', got %q", body)
		}
	}

	// individual unhealthy check
	{
		uri := "http://0.0.0.0:8281/readyz/redis"
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, uri, http.NoBody)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 503 {
			t.Errorf("/readyz/redis: expected 503, got %d", resp.StatusCode)
		}
		if !strings.Contains(string(body), "[-]redis failed") {
			t.Errorf("expected '[-]redis failed', got %q", body)
		}
	}

	// individual check not found
	{
		uri := "http://0.0.0.0:8281/livez/nonexistent"
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, uri, http.NoBody)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 404 {
			t.Errorf("/livez/nonexistent: expected 404, got %d", resp.StatusCode)
		}
	}
}

func TestVerbose(t *testing.T) {
	reporter := httpserver.NewReporter(httpserver.Config{
		Addr:           "0.0.0.0",
		Port:           8282,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		defer cancel()
		reporter.Stop(ctx)
	})

	if err := reporter.Run(ctx); err != nil {
		t.Fatal(err)
	}

	reporter.SetLiveness(ctx, true)
	reporter.UpdateHealthChecks(ctx, map[string]*health.CheckResult{
		"postgres": {Name: "postgres", Status: health.StatusHealthy},
		"redis":    {Name: "redis", Status: health.StatusUnhealthy, Error: errors.New("timeout")},
	})

	client := http.Client{Timeout: time.Second}

	// verbose lists all checks
	{
		uri := "http://0.0.0.0:8282/livez?verbose"
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, uri, http.NoBody)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		text := string(body)

		if resp.StatusCode != 503 {
			t.Errorf("verbose with failing check: expected 503, got %d", resp.StatusCode)
		}
		if !strings.Contains(text, "[+]postgres ok") {
			t.Errorf("expected postgres ok in verbose, got %q", text)
		}
		if !strings.Contains(text, "[-]redis failed") {
			t.Errorf("expected redis failed in verbose, got %q", text)
		}
	}

	// verbose with exclude
	{
		uri := "http://0.0.0.0:8282/livez?verbose&exclude=redis"
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, uri, http.NoBody)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		text := string(body)

		if resp.StatusCode != 200 {
			t.Errorf("verbose excluding failing check: expected 200, got %d", resp.StatusCode)
		}
		if strings.Contains(text, "redis") {
			t.Errorf("excluded redis should not appear in output, got %q", text)
		}
		if !strings.Contains(text, "[+]postgres ok") {
			t.Errorf("expected postgres in output, got %q", text)
		}
	}
}
