package infrastructure

type NoopLogger struct{}

func (n NoopLogger) Infof(string, ...interface{}) {}

func (n NoopLogger) Errorf(string, ...interface{}) {}
