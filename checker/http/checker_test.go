package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/schigh/health/v2"
	httpchecker "github.com/schigh/health/v2/checker/http"
)

func TestChecker_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := httpchecker.NewChecker("test", srv.URL)
	result := c.Check(context.Background())

	if result.Status != health.StatusHealthy {
		t.Fatalf("expected healthy, got %s (err: %v)", result.Status, result.Error)
	}
	if result.Duration == 0 {
		t.Fatal("expected non-zero duration")
	}
}

func TestChecker_WrongStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := httpchecker.NewChecker("test", srv.URL)
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Fatalf("expected unhealthy, got %s", result.Status)
	}
}

func TestChecker_ExpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := httpchecker.NewChecker("test", srv.URL, httpchecker.WithExpectedStatus(http.StatusNoContent))
	result := c.Check(context.Background())

	if result.Status != health.StatusHealthy {
		t.Fatalf("expected healthy with 204, got %s (err: %v)", result.Status, result.Error)
	}
}

func TestChecker_ConnectionRefused(t *testing.T) {
	c := httpchecker.NewChecker("test", "http://127.0.0.1:1", httpchecker.WithTimeout(100*time.Millisecond))
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Fatalf("expected unhealthy for refused connection, got %s", result.Status)
	}
}

func TestChecker_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := httpchecker.NewChecker("test", srv.URL, httpchecker.WithTimeout(50*time.Millisecond))
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Fatalf("expected unhealthy on timeout, got %s", result.Status)
	}
}

func TestChecker_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := httpchecker.NewChecker("test", srv.URL)
	result := c.Check(ctx)

	if result.Status != health.StatusUnhealthy {
		t.Fatalf("expected unhealthy on cancelled context, got %s", result.Status)
	}
}
