package infra

type NoopLogger struct{}

func (n NoopLogger) Infof(_ string, _ ...any) {
	// Interface implementation
}

func (n NoopLogger) Errorf(_ string, _ ...any) {
	// Interface implementation
}
