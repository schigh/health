package health

type Config struct {
	HTTPPort          int    `envconfig:"HEALTH_HTTP_PORT" default:"8181"`
	LivenessEndpoint  string `envconfig:"HEALTH_LIVENESS_ENDPOINT" default:"/live"`
	ReadinessEndpoint string `envconfig:"HEALTH_READINESS_ENDPOINT" default:"/ready"`
}
