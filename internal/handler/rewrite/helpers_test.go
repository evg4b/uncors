package rewrite_test

import (
	"context"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/stretchr/testify/assert"
)

func TestGetRewriteHost(t *testing.T) {
	t.Run("returns host when exists", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), rewrite.RewriteHostKey, "example.com")
		request := &contracts.Request{}

		assert.Equal(t, "example.com", rewrite.GetRewriteHost(request.WithContext(ctx)))
	})

	t.Run("returns empty string when host is not set", func(t *testing.T) {
		request := &contracts.Request{}

		assert.Empty(t, rewrite.GetRewriteHost(request.WithContext(t.Context())))
	})
}
