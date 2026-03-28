package httpserver

import "github.com/schigh/health"

type Config struct {
	Addr           string `envconfig:"HEALTH_HTTP_REPORTER_ADDR" default:"0.0.0.0"`
	Port           int    `envconfig:"HEALTH_HTTP_REPORTER_PORT" default:"8181"`
	LivenessRoute  string `envconfig:"HEALTH_HTTP_REPORTER_LIVENESS_ROUTE" default:"/live"`
	ReadinessRoute string `envconfig:"HEALTH_HTTP_REPORTER_READINESS_ROUTE" default:"/ready"`
	StartupRoute   string `envconfig:"HEALTH_HTTP_REPORTER_STARTUP_ROUTE" default:"/startup"`
	Logger         health.Logger
}
