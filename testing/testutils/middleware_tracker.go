package testutils

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/evg4b/uncors/testing/mocks"
)

type MidelwaresTracker struct {
	CallsOrder []string
	t          *testing.T
}

func NewMidelwaresTracker(t *testing.T) *MidelwaresTracker {
	t.Helper()

	return &MidelwaresTracker{
		CallsOrder: []string{},
		t:          t,
	}
}

func (t *MidelwaresTracker) MakeMidelware(name string) *mocks.HandlingMiddlewareMock {
	t.t.Helper()

	return mocks.NewHandlingMiddlewareMock(t.t).WrapMock.
		Set(func(next infrastructure.HandlerFunc) infrastructure.HandlerFunc {
			t.t.Helper()

			return func(w http.ResponseWriter, r *http.Request) error {
				t.t.Helper()
				t.CallsOrder = append(t.CallsOrder, name)

				return next(w, r)
			}
		})
}

func (t *MidelwaresTracker) MakeFinalMidelware(name string) *mocks.HandlingMiddlewareMock {
	t.t.Helper()

	return mocks.NewHandlingMiddlewareMock(t.t).WrapMock.
		Set(func(next infrastructure.HandlerFunc) infrastructure.HandlerFunc {
			t.t.Helper()

			return func(w http.ResponseWriter, r *http.Request) error {
				t.t.Helper()
				t.CallsOrder = append(t.CallsOrder, name)

				return nil
			}
		})
}
