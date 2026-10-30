package helpers_test

import (
	"net/url"
	"testing"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeLogValue(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		expected string
	}{
		{name: "plain value", in: "value", expected: "value"},
		{name: "line feed", in: "a\nb", expected: "a b"},
		{name: "carriage return", in: "a\rb", expected: "a b"},
		{name: "forged entry", in: "x\r\nlevel=ERROR msg=forged", expected: "x  level=ERROR msg=forged"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, helpers.SanitizeLogValue(testCase.in))
		})
	}
}

func TestLogURL(t *testing.T) {
	t.Run("drops user info", func(t *testing.T) {
		parsed, err := url.Parse("https://user:secret@api.local:8443/path?q=1#top")
		require.NoError(t, err)

		logged := helpers.LogURL(parsed)

		assert.Equal(t, "https://api.local:8443/path?q=1#top", logged)
		assert.NotContains(t, logged, "secret")
		assert.NotContains(t, logged, "user")
	})

	t.Run("keeps an encoded path", func(t *testing.T) {
		parsed, err := url.Parse("http://api.local/a%2Fb")
		require.NoError(t, err)

		assert.Equal(t, "http://api.local/a%2Fb", helpers.LogURL(parsed))
	})

	t.Run("nil", func(t *testing.T) {
		assert.Empty(t, helpers.LogURL(nil))
	})
}
