package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/schigh/health/v2"
)

const DefaultTimeout = 5 * time.Second

// Checker performs HTTP health checks against a URL endpoint.
type Checker struct {
	name           string
	url            string
	client         *http.Client
	timeout        time.Duration
	expectedStatus int
	method         string
}

// Option is a functional option for configuring an HTTP Checker.
type Option func(*Checker)

// WithTimeout sets the timeout for the HTTP request.
func WithTimeout(d time.Duration) Option {
	return func(c *Checker) { c.timeout = d }
}

// WithExpectedStatus sets the expected HTTP status code. Default is 200.
func WithExpectedStatus(code int) Option {
	return func(c *Checker) { c.expectedStatus = code }
}

// WithMethod sets the HTTP method. Default is GET.
func WithMethod(method string) Option {
	return func(c *Checker) { c.method = method }
}

// WithClient sets a custom http.Client.
func WithClient(client *http.Client) Option {
	return func(c *Checker) { c.client = client }
}

// NewChecker returns an HTTP health checker for the given URL.
func NewChecker(name, url string, opts ...Option) *Checker {
	c := &Checker{
		name:           name,
		url:            url,
		timeout:        DefaultTimeout,
		expectedStatus: http.StatusOK,
		method:         http.MethodGet,
	}
	for _, o := range opts {
		o(c)
	}
	if c.client == nil {
		c.client = &http.Client{Timeout: c.timeout}
	}
	return c
}

func (c *Checker) Check(ctx context.Context) *health.CheckResult {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, c.method, c.url, http.NoBody)
	if err != nil {
		return &health.CheckResult{
			Name:      c.name,
			Status:    health.StatusUnhealthy,
			Error:     fmt.Errorf("create request: %w", err),
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return &health.CheckResult{
			Name:      c.name,
			Status:    health.StatusUnhealthy,
			Error:     fmt.Errorf("request failed: %w", err),
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != c.expectedStatus {
		return &health.CheckResult{
			Name:      c.name,
			Status:    health.StatusUnhealthy,
			Error:     fmt.Errorf("expected status %d, got %d", c.expectedStatus, resp.StatusCode),
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}

	return &health.CheckResult{
		Name:      c.name,
		Status:    health.StatusHealthy,
		Duration:  time.Since(start),
		Timestamp: start,
	}
}
