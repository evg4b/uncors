package mocks

import (
	"github.com/evg4b/uncors/internal/processor"
	"net/http"
	"testing"
)

type MiddlewaresTracker struct {
	CallsOrder []string
	t          *testing.T
}

func NewMiddlewaresTracker(t *testing.T) *MiddlewaresTracker {
	t.Helper()

	return &MiddlewaresTracker{
		CallsOrder: []string{},
		t:          t,
	}
}

func (t *MiddlewaresTracker) MakeMiddleware(name string) *HandlingMiddlewareMock {
	t.t.Helper()

	return NewHandlingMiddlewareMock(t.t).WrapMock.
		Set(func(next processor.HandlerFunc) processor.HandlerFunc {
			t.t.Helper()

			return func(w http.ResponseWriter, r *http.Request) error {
				t.t.Helper()
				t.CallsOrder = append(t.CallsOrder, name)

				return next(w, r)
			}
		})
}

func (t *MiddlewaresTracker) MakeFinalMiddleware(name string) *HandlingMiddlewareMock {
	t.t.Helper()

	return NewHandlingMiddlewareMock(t.t).WrapMock.
		Set(func(next processor.HandlerFunc) processor.HandlerFunc {
			t.t.Helper()

			return func(w http.ResponseWriter, r *http.Request) error {
				t.t.Helper()
				t.CallsOrder = append(t.CallsOrder, name)

				return nil
			}
		})
}
