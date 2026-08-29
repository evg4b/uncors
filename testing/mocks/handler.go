package mocks

import (
	"net/http"
	"sync/atomic"
	"testing"
)

// FailNowHandlerMock fails the test if the request reaches it.
func FailNowHandlerMock(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("should not be called")
	})
}

// HandlerMock is an http.Handler that records how often it was called.
type HandlerMock struct {
	handler http.Handler
	calls   atomic.Uint64
}

func NewHandlerMock(handler http.Handler) *HandlerMock {
	if handler == nil {
		handler = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	}

	return &HandlerMock{handler: handler}
}

func (m *HandlerMock) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	m.calls.Add(1)
	m.handler.ServeHTTP(writer, request)
}

func (m *HandlerMock) Calls() uint64 {
	return m.calls.Load()
}
