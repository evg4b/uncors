package rewrite_test

import (
	"context"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRewriteRequest(t *testing.T) {
	t.Run("returns true when rewrite host exists", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), rewrite.RewriteHostKey, "example.com")
		request := &contracts.Request{}

		result := rewrite.IsRewriteRequest(
			request.WithContext(ctx),
		)

		assert.True(t, result)
	})

	t.Run("returns false when rewrite host is not set", func(t *testing.T) {
		request := &contracts.Request{}

		result := rewrite.IsRewriteRequest(
			request.WithContext(context.Background()),
		)

		assert.False(t, result)
	})
}

func TestGetRewriteHost(t *testing.T) {
	t.Run("returns host when exists", func(t *testing.T) {
		expected := "example.com"
		ctx := context.WithValue(context.Background(), rewrite.RewriteHostKey, expected)
		request := &contracts.Request{}

		result, err := rewrite.GetRewriteHost(
			request.WithContext(ctx),
		)

		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("returns empty string when host is not set", func(t *testing.T) {
		request := &contracts.Request{}

		result, err := rewrite.GetRewriteHost(
			request.WithContext(context.Background()),
		)

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("returns error when host has invalid type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), rewrite.RewriteHostKey, 123)
		request := &contracts.Request{}

		result, err := rewrite.GetRewriteHost(
			request.WithContext(ctx),
		)

		require.ErrorIs(t, err, rewrite.ErrInvalidHost)
		assert.Empty(t, result)
	})
}
