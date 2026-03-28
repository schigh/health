package redis_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/schigh/health/v2"
	"github.com/schigh/health/v2/checker/redis"
)

// mockRedisServer accepts one connection and responds to RESP commands.
func mockRedisServer(t *testing.T, handler func(net.Conn)) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		handler(conn)
	}()
	return ln
}

func TestChecker_Healthy(t *testing.T) {
	ln := mockRedisServer(t, func(conn net.Conn) {
		buf := make([]byte, 256)
		n, _ := conn.Read(buf)
		_ = n
		fmt.Fprint(conn, "+PONG\r\n")
	})
	defer ln.Close()

	c := redis.NewChecker("test", ln.Addr().String(), redis.WithTimeout(time.Second))
	result := c.Check(context.Background())

	if result.Status != health.StatusHealthy {
		t.Fatalf("expected healthy, got %s (err: %v)", result.Status, result.Error)
	}
}

func TestChecker_AuthSuccess(t *testing.T) {
	ln := mockRedisServer(t, func(conn net.Conn) {
		buf := make([]byte, 256)
		// read AUTH command
		conn.Read(buf)
		fmt.Fprint(conn, "+OK\r\n")
		// read PING command
		conn.Read(buf)
		fmt.Fprint(conn, "+PONG\r\n")
	})
	defer ln.Close()

	c := redis.NewChecker("test", ln.Addr().String(),
		redis.WithPassword("secret"),
		redis.WithTimeout(time.Second),
	)
	result := c.Check(context.Background())

	if result.Status != health.StatusHealthy {
		t.Fatalf("expected healthy with auth, got %s (err: %v)", result.Status, result.Error)
	}
}

func TestChecker_AuthFailure(t *testing.T) {
	ln := mockRedisServer(t, func(conn net.Conn) {
		buf := make([]byte, 256)
		conn.Read(buf)
		fmt.Fprint(conn, "-ERR invalid password\r\n")
	})
	defer ln.Close()

	c := redis.NewChecker("test", ln.Addr().String(),
		redis.WithPassword("wrong"),
		redis.WithTimeout(time.Second),
	)
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Fatalf("expected unhealthy on auth failure, got %s", result.Status)
	}
}

func TestChecker_ConnectionRefused(t *testing.T) {
	c := redis.NewChecker("test", "127.0.0.1:1", redis.WithTimeout(100*time.Millisecond))
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Fatalf("expected unhealthy, got %s", result.Status)
	}
}

func TestChecker_MalformedResponse(t *testing.T) {
	ln := mockRedisServer(t, func(conn net.Conn) {
		buf := make([]byte, 256)
		conn.Read(buf)
		fmt.Fprint(conn, "garbage\r\n")
	})
	defer ln.Close()

	c := redis.NewChecker("test", ln.Addr().String(), redis.WithTimeout(time.Second))
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Fatalf("expected unhealthy on malformed response, got %s", result.Status)
	}
}
