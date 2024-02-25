package db

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	healthpb "github.com/schigh/health/pkg/v1"
)

const (
	DefaultPingTimeout = 3 * time.Second
)

// CtxPinger defines the interface for the database
// functionality used to perform the health check.
type CtxPinger interface {
	PingContext(context.Context) error
}

// Checker implements health.Checker.
type Checker struct {
	name    string
	pinger  CtxPinger
	timeout time.Duration
}

// Option is a functional decorator for creating a new Checker.
type Option func(*Checker)

// WithTimeout sets the timeout on the PingContext check.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Checker) {
		c.timeout = timeout
	}
}

// NewChecker returns a Checker using the provided name and CtxPinger.
func NewChecker(name string, pinger CtxPinger, opts ...Option) *Checker {
	out := Checker{
		name:    name,
		pinger:  pinger,
		timeout: DefaultPingTimeout,
	}

	for i := range opts {
		opts[i](&out)
	}

	return &out
}

func (c *Checker) Check(ctx context.Context) *healthpb.Check {
	now := time.Now()
	select {
	case <-ctx.Done():
		return &healthpb.Check{
			Name:    c.name,
			Healthy: false,
			Error: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"origin": structpb.NewStringValue("github.com/schigh/health/checker/db.Checker.Check"),
					"text":   structpb.NewStringValue("invalid context"),
				},
			},
			ErrorSince: timestamppb.New(now),
		}
	default:
	}

	out := healthpb.Check{
		Name:    c.name,
		Healthy: true,
	}
	if c.pinger == nil {
		out.Healthy = false
		out.Error = &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"origin": structpb.NewStringValue("github.com/schigh/health/checker/db.Checker.Check"),
				"text":   structpb.NewStringValue("invalid context"),
			},
		}
		out.ErrorSince = timestamppb.New(now)
		return &out
	}

	cCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	err := c.pinger.PingContext(cCtx)
	if err != nil {
		out.Healthy = false
		out.Error = &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"origin": structpb.NewStringValue("github.com/schigh/health/checker/db.Checker.Check"),
				"text":   structpb.NewStringValue(err.Error()),
			},
		}
		out.ErrorSince = timestamppb.New(now)
	}

	return &out
}
