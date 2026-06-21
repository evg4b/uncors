package mocks

import (
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infra"
)

func FailNowHandlerMock(t *testing.T) contracts.Handler {
	return infra.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) error {
		t.Fatal("should not be called")

		return nil
	})
}
