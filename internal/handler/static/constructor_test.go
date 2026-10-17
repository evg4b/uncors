package static_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/handler/static"
	"github.com/stretchr/testify/assert"
)

func TestNewMiddleware_RequiresFileSystem(t *testing.T) {
	assert.PanicsWithValue(
		t,
		"StaticMiddleware: File system is not configured",
		func() { static.NewMiddleware() },
	)
}
