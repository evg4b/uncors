package infra

type NoopLogger struct{}

func (n NoopLogger) Infof(string, ...any) {
	// Interface implementation
}

func (n NoopLogger) Errorf(string, ...any) {
	// Interface implementation
}
