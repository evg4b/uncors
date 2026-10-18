package rewrite_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/stretchr/testify/assert"
)

func TestNewMiddleware_RequiresRewritingOptions(t *testing.T) {
	assert.PanicsWithValue(
		t,
		"RewriteMiddleware: Rewriting options are not configured",
		func() { rewrite.NewMiddleware() },
	)
}
