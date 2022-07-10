package testutils

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/infrastrucure"
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
		Set(func(next infrastrucure.HandlerFunc) (h1 infrastrucure.HandlerFunc) {
			return func(w http.ResponseWriter, r *http.Request) {
				t.CallsOrder = append(t.CallsOrder, name)
				next(w, r)
			}
		})
}

func (t *midelwaresTracker) MakeFinalMidelware(name string) processor.HandlingMiddleware {
	return mocks.NewHandlingMiddlewareMock(t.t).WrapMock.
		Set(func(next infrastrucure.HandlerFunc) (h1 infrastrucure.HandlerFunc) {
			return func(w http.ResponseWriter, r *http.Request) {
				t.CallsOrder = append(t.CallsOrder, name)
			}
		})
}
