package health

import "testing"

func TestNoOpLogger_Methods(t *testing.T) {
	var l NoOpLogger
	l.Debug("msg", "key", "val")
	l.Info("msg", "key", "val")
	l.Warn("msg", "key", "val")
	l.Error("msg", "key", "val")
}
