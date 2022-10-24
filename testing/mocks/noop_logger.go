package mocks

import "testing"

func NewNoopLogger(t *testing.T) *LoggerMock {
	return NewLoggerMock(t).
		ErrorfMock.Return().
		ErrorfMock.Return().
		WarningMock.Return().
		WarningfMock.Return().
		InfoMock.Return().
		InfofMock.Return().
		DebugMock.Return().
		DebugfMock.Return().
		PrintResponseMock.Return()
}
