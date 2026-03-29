package redis

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"time"

	"github.com/schigh/health/v2"
)

const DefaultTimeout = 5 * time.Second

// Checker performs Redis health checks using the raw RESP protocol.
// Zero external dependencies. Supports standalone Redis with optional
// legacy AUTH. Does not support Redis Cluster or ACL-only (Redis 6+
// without legacy password).
type Checker struct {
	name     string
	addr     string
	timeout  time.Duration
	password string
}

// Option is a functional option for configuring a Redis Checker.
type Option func(*Checker)

// WithTimeout sets the dial and read/write timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Checker) { c.timeout = d }
}

// WithPassword sets the password for legacy AUTH before PING.
// This sends AUTH <password> in cleartext over TCP. For production
// Redis instances requiring authentication, use TLS.
func WithPassword(password string) Option {
	return func(c *Checker) { c.password = password }
}

// NewChecker returns a Redis health checker for the given address (host:port).
func NewChecker(name, addr string, opts ...Option) *Checker {
	c := &Checker{name: name, addr: addr, timeout: DefaultTimeout}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *Checker) Check(ctx context.Context) *health.CheckResult {
	start := time.Now()

	var d net.Dialer
	d.Timeout = c.timeout

	conn, err := d.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return unhealthy(c.name, start, fmt.Errorf("dial %s: %w", c.addr, err))
	}
	defer conn.Close()

	deadline := time.Now().Add(c.timeout)
	_ = conn.SetDeadline(deadline)

	reader := bufio.NewReader(conn)

	if c.password != "" {
		if err := c.authenticate(conn, reader); err != nil {
			return unhealthy(c.name, start, err)
		}
	}

	if err := c.ping(conn, reader); err != nil {
		return unhealthy(c.name, start, err)
	}

	return &health.CheckResult{
		Name:      c.name,
		Status:    health.StatusHealthy,
		Duration:  time.Since(start),
		Timestamp: start,
	}
}

// authenticate sends AUTH and validates the response.
func (c *Checker) authenticate(conn net.Conn, reader *bufio.Reader) error {
	_, err := fmt.Fprintf(conn, "*2\r\n$4\r\nAUTH\r\n$%d\r\n%s\r\n", len(c.password), c.password)
	if err != nil {
		return fmt.Errorf("write AUTH: %w", err)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read AUTH response: %w", err)
	}
	if len(line) < 3 || line[0] != '+' {
		return fmt.Errorf("AUTH failed: %s", line)
	}
	return nil
}

// ping sends PING and validates the PONG response.
func (c *Checker) ping(conn net.Conn, reader *bufio.Reader) error {
	_, err := fmt.Fprint(conn, "*1\r\n$4\r\nPING\r\n")
	if err != nil {
		return fmt.Errorf("write PING: %w", err)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read PING response: %w", err)
	}
	if len(line) < 5 || line[:5] != "+PONG" {
		return fmt.Errorf("unexpected PING response: %s", line)
	}
	return nil
}

func unhealthy(name string, start time.Time, err error) *health.CheckResult {
	return &health.CheckResult{
		Name:      name,
		Status:    health.StatusUnhealthy,
		Error:     err,
		ErrorSince: start,
		Duration:  time.Since(start),
		Timestamp: start,
	}
}
