package testutils

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/evg4b/uncors/internal/processor"
	"github.com/evg4b/uncors/testing/mocks"
)

type midelwaresTracker struct {
	CallsOrder []string
	t          *testing.T
}

func NewMidelwaresTracker(t *testing.T) *midelwaresTracker {
	return &midelwaresTracker{
		CallsOrder: []string{},
		t:          t,
	}
}

func (t *midelwaresTracker) MakeMidelware(name string) processor.HandlingMiddleware {
	return mocks.NewHandlingMiddlewareMock(t.t).WrapMock.
		Set(func(next infrastructure.HandlerFunc) (h1 infrastructure.HandlerFunc) {
			return func(w http.ResponseWriter, r *http.Request) error {
				t.CallsOrder = append(t.CallsOrder, name)
				return next(w, r)
			}
		})
}

func (t *midelwaresTracker) MakeFinalMidelware(name string) processor.HandlingMiddleware {
	return mocks.NewHandlingMiddlewareMock(t.t).WrapMock.
		Set(func(next infrastructure.HandlerFunc) (h1 infrastructure.HandlerFunc) {
			return func(w http.ResponseWriter, r *http.Request) error {
				t.CallsOrder = append(t.CallsOrder, name)
				return nil
			}
		})
}
