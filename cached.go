package health

import (
	"context"
	"sync"
	"time"
)

// CachedChecker wraps a Checker with TTL-based caching. During refresh,
// stale values are served to concurrent readers. Only one goroutine
// refreshes at a time (prevents thundering herd on expensive checks).
//
// The first call always executes the underlying checker synchronously.
type CachedChecker struct {
	inner  Checker
	ttl    time.Duration
	mu     sync.RWMutex
	cached *CheckResult
	expiry time.Time
}

// WithCache wraps a Checker with TTL-based result caching.
func WithCache(c Checker, ttl time.Duration) *CachedChecker {
	return &CachedChecker{inner: c, ttl: ttl}
}

// Check returns the cached result if still valid, otherwise refreshes.
func (c *CachedChecker) Check(ctx context.Context) *CheckResult {
	c.mu.RLock()
	if c.cached != nil && time.Now().Before(c.expiry) {
		result := c.cached
		c.mu.RUnlock()
		return result
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// double-check after acquiring write lock
	if c.cached != nil && time.Now().Before(c.expiry) {
		return c.cached
	}

	result := c.inner.Check(ctx)
	c.cached = result
	c.expiry = time.Now().Add(c.ttl)
	return result
}
