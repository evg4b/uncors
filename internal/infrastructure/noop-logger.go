package infrastructure

type NoopLogger struct{}

func (n NoopLogger) Infof(format string, v ...interface{}) {}

func (n NoopLogger) Errorf(format string, v ...interface{}) {}
