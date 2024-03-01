package health

// Logger defines the logging interface used internally.
// This interface is implemented by log/slog, but can be
// easily adapted from other loggers.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// NoOpLogger is used to suppress log output.
type NoOpLogger struct{}

func (n NoOpLogger) Debug(_ string, _ ...any) {}
func (n NoOpLogger) Info(_ string, _ ...any)  {}
func (n NoOpLogger) Warn(_ string, _ ...any)  {}
func (n NoOpLogger) Error(_ string, _ ...any) {}
