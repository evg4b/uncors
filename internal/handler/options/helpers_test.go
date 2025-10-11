package options

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetHeaderOrDefault(t *testing.T) {
	t.Run("sets value when not empty", func(t *testing.T) {
		header := http.Header{}
		SetHeaderOrDefault(header, "X-Test-Header", "test-value", "default-value")
		assert.Equal(t, "test-value", header.Get("X-Test-Header"))
	})

	t.Run("sets default when value is empty", func(t *testing.T) {
		header := http.Header{}
		SetHeaderOrDefault(header, "X-Test-Header", "", "default-value")
		assert.Equal(t, "default-value", header.Get("X-Test-Header"))
	})

	t.Run("sets default when value is empty string", func(t *testing.T) {
		header := http.Header{}
		SetHeaderOrDefault(header, "X-Test-Header", "", "*")
		assert.Equal(t, "*", header.Get("X-Test-Header"))
	})

	t.Run("overwrites existing header with value", func(t *testing.T) {
		header := http.Header{}
		header.Set("X-Test-Header", "old-value")
		SetHeaderOrDefault(header, "X-Test-Header", "new-value", "default-value")
		assert.Equal(t, "new-value", header.Get("X-Test-Header"))
	})

	t.Run("overwrites existing header with default", func(t *testing.T) {
		header := http.Header{}
		header.Set("X-Test-Header", "old-value")
		SetHeaderOrDefault(header, "X-Test-Header", "", "default-value")
		assert.Equal(t, "default-value", header.Get("X-Test-Header"))
	})
}
