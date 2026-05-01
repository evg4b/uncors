package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestRequestMatcherIsPathOnly(t *testing.T) {
	t.Run("returns true when only path is set", func(t *testing.T) {
		m := &config.RequestMatcher{Path: "/api/resource"}
		assert.True(t, m.IsPathOnly())
	})

	t.Run("returns false when method is set", func(t *testing.T) {
		m := &config.RequestMatcher{Path: "/api", Method: "GET"}
		assert.False(t, m.IsPathOnly())
	})

	t.Run("returns false when queries are set", func(t *testing.T) {
		m := &config.RequestMatcher{
			Path:    "/api",
			Queries: map[string]string{"key": "value"},
		}
		assert.False(t, m.IsPathOnly())
	})

	t.Run("returns false when headers are set", func(t *testing.T) {
		m := &config.RequestMatcher{
			Path:    "/api",
			Headers: map[string]string{"X-Token": "abc"},
		}
		assert.False(t, m.IsPathOnly())
	})

	t.Run("returns true when nothing is set", func(t *testing.T) {
		m := &config.RequestMatcher{}
		assert.True(t, m.IsPathOnly())
	})
}
