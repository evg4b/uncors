package mocks

import (
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
)

func FailNowMock(t *testing.T) contracts.Handler {
	return contracts.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) {
		t.Fatal("should not be called")
	})
}
