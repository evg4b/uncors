package mock_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewHandler_RequiresResponse(t *testing.T) {
	assert.PanicsWithValue(
		t,
		"MockHandler: Response is not configured",
		func() { mock.NewHandler() },
	)
}
