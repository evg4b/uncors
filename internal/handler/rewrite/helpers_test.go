package rewrite_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRewriteHost(t *testing.T) {
	t.Run("returns host when exists", func(t *testing.T) {
		expected := "example.com"
		ctx := context.WithValue(t.Context(), rewrite.RewriteHostKey, expected)
		request := &http.Request{}

		result, err := rewrite.GetRewriteHost(
			request.WithContext(ctx),
		)

		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("returns empty string when host is not set", func(t *testing.T) {
		request := &http.Request{}

		result, err := rewrite.GetRewriteHost(
			request.WithContext(t.Context()),
		)

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("returns error when host has invalid type", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), rewrite.RewriteHostKey, 123)
		request := &http.Request{}

		result, err := rewrite.GetRewriteHost(
			request.WithContext(ctx),
		)

		require.ErrorIs(t, err, rewrite.ErrInvalidHost)
		assert.Empty(t, result)
	})
}
