package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowedOriginsClone(t *testing.T) {
	origins := config.AllowedOrigins{"http://localhost:*", "https://*.example.test"}

	cloned := origins.Clone()

	assert.Equal(t, origins, cloned)
	assert.NotSame(t, &origins[0], &cloned[0])
}

func TestAllowedOriginsAllows(t *testing.T) {
	origins := config.AllowedOrigins{
		"http://localhost:*",
		"https://*.example.test",
		"http://127.0.0.1:{3000,5173}",
	}

	tests := []struct {
		name     string
		origins  config.AllowedOrigins
		origin   string
		expected bool
	}{
		{name: "any origin without patterns", origins: nil, origin: "https://attacker.test", expected: true},
		{name: "request without origin", origins: origins, origin: "", expected: true},
		{name: "port wildcard", origins: origins, origin: "http://localhost:3000", expected: true},
		{name: "subdomain wildcard", origins: origins, origin: "https://app.example.test", expected: true},
		{name: "alternatives", origins: origins, origin: "http://127.0.0.1:5173", expected: true},
		{name: "other port", origins: origins, origin: "http://127.0.0.1:8080", expected: false},
		{name: "other scheme", origins: origins, origin: "https://localhost:3000", expected: false},
		{name: "other host", origins: origins, origin: "https://attacker.test", expected: false},
		{name: "suffix of an allowed host", origins: origins, origin: "https://example.test.attacker.test", expected: false},
		{name: "opaque origin", origins: origins, origin: "null", expected: false},
		{name: "exact origin", origins: config.AllowedOrigins{"null"}, origin: "null", expected: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.origins.Allows(test.origin))
		})
	}
}

func TestAllowedOriginsValidate(t *testing.T) {
	t.Run("accepts valid patterns", func(t *testing.T) {
		require.NoError(t, config.AllowedOrigins{
			"http://localhost:*",
			"https://*.example.test",
			"http://127.0.0.1:{3000,5173}",
			"null",
		}.Validate("allowed-origins"))
	})

	t.Run("accepts no patterns", func(t *testing.T) {
		require.NoError(t, config.AllowedOrigins(nil).Validate("allowed-origins"))
	})

	tests := []struct {
		name    string
		pattern string
		error   string
	}{
		{name: "empty", pattern: "", error: "allowed-origins[0] must not be empty"},
		{name: "invalid glob", pattern: "http://[localhost", error: "allowed-origins[0] is not a valid glob pattern"},
		{
			name:    "trailing slash",
			pattern: "http://localhost:3000/",
			error:   "allowed-origins[0] must not end with a slash: an origin has no path",
		},
	}

	for _, test := range tests {
		t.Run("rejects "+test.name, func(t *testing.T) {
			require.EqualError(t, config.AllowedOrigins{test.pattern}.Validate("allowed-origins"), test.error)
		})
	}

	t.Run("reports every invalid pattern", func(t *testing.T) {
		err := config.AllowedOrigins{"", "http://localhost:*", "http://[x"}.Validate("allowed-origins")

		require.EqualError(t, err,
			"allowed-origins[0] must not be empty\nallowed-origins[2] is not a valid glob pattern")
	})
}
