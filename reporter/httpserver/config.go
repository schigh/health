package httpserver

import "github.com/schigh/health/v2"

// Config holds the configuration for the HTTP reporter.
type Config struct {
	Addr           string
	Port           int
	LivenessRoute  string
	ReadinessRoute string
	StartupRoute   string
	Logger         health.Logger
	Middleware     []Middleware
	ServiceName    string
	ServiceVersion string
}

// Option is a functional option for configuring the HTTP reporter.
type Option func(*Config)

// WithAddr sets the bind address. Default: "0.0.0.0".
func WithAddr(addr string) Option {
	return func(c *Config) { c.Addr = addr }
}

// WithPort sets the listen port. Default: 8181.
func WithPort(port int) Option {
	return func(c *Config) { c.Port = port }
}

// WithLivenessRoute sets the liveness endpoint path. Default: "/live".
func WithLivenessRoute(route string) Option {
	return func(c *Config) { c.LivenessRoute = route }
}

// WithReadinessRoute sets the readiness endpoint path. Default: "/ready".
func WithReadinessRoute(route string) Option {
	return func(c *Config) { c.ReadinessRoute = route }
}

// WithStartupRoute sets the startup endpoint path. Default: "/startup".
func WithStartupRoute(route string) Option {
	return func(c *Config) { c.StartupRoute = route }
}

// WithLogger sets the logger. Default: NoOpLogger.
func WithLogger(l health.Logger) Option {
	return func(c *Config) { c.Logger = l }
}

// WithMiddleware adds middleware to the reporter's handler chain.
// The first middleware in the list is the first to see the request.
func WithMiddleware(mw ...Middleware) Option {
	return func(c *Config) { c.Middleware = append(c.Middleware, mw...) }
}

// WithServiceName sets the service name for the /.well-known/health manifest.
func WithServiceName(name string) Option {
	return func(c *Config) { c.ServiceName = name }
}

// WithServiceVersion sets the service version for the /.well-known/health manifest.
func WithServiceVersion(version string) Option {
	return func(c *Config) { c.ServiceVersion = version }
}

func defaultConfig() Config {
	return Config{
		Addr:           "0.0.0.0",
		Port:           8181,
		LivenessRoute:  "/livez",
		ReadinessRoute: "/readyz",
		StartupRoute:   "/healthz",
	}
}
