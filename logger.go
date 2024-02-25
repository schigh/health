package health

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type NoOpLogger struct{}

func (n NoOpLogger) Debug(_ string, _ ...any) {}
func (n NoOpLogger) Info(_ string, _ ...any)  {}
func (n NoOpLogger) Warn(_ string, _ ...any)  {}
func (n NoOpLogger) Error(_ string, _ ...any) {}
