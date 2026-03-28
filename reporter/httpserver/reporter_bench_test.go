package httpserver_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/schigh/health/v2"
	"github.com/schigh/health/v2/reporter/httpserver"
)

func BenchmarkReportLiveness(b *testing.B) {
	reporter := httpserver.NewReporter(httpserver.Config{
		Addr:           "0.0.0.0",
		Port:           18181,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})

	ctx := context.Background()
	reporter.Run(ctx)
	defer reporter.Stop(ctx)

	reporter.SetLiveness(ctx, true)
	reporter.UpdateHealthChecks(ctx, map[string]*health.CheckResult{
		"postgres": {Name: "postgres", Status: health.StatusHealthy, Group: "database", ComponentType: "datastore"},
		"redis":    {Name: "redis", Status: health.StatusHealthy, Group: "cache", ComponentType: "datastore"},
		"payments": {Name: "payments", Status: health.StatusHealthy, Group: "services", ComponentType: "http"},
	})

	req := httptest.NewRequest(http.MethodGet, "/livez", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		reporter.Recover(http.HandlerFunc(func(rw http.ResponseWriter, rq *http.Request) {
			// simulate the handler path without going through the full server
		})).ServeHTTP(w, req)
	}
}

func BenchmarkCacheHealthChecks(b *testing.B) {
	reporter := httpserver.NewReporter(httpserver.Config{
		Addr:           "0.0.0.0",
		Port:           18182,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	})

	ctx := context.Background()
	reporter.Run(ctx)
	defer reporter.Stop(ctx)

	checks := map[string]*health.CheckResult{
		"postgres": {Name: "postgres", Status: health.StatusHealthy, Group: "database", ComponentType: "datastore"},
		"redis":    {Name: "redis", Status: health.StatusHealthy, Group: "cache", ComponentType: "datastore"},
		"payments": {Name: "payments", Status: health.StatusHealthy, Group: "services", ComponentType: "http"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reporter.UpdateHealthChecks(ctx, checks)
	}
}
