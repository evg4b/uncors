package infra

type NoopLogger struct{}

func (n NoopLogger) Infof(string, ...any) {}

func (n NoopLogger) Errorf(string, ...any) {}
