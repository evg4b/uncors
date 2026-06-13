package urlt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyDefaultScheme(t *testing.T) {
	t.Run("no default scheme", func(t *testing.T) {
		tests := []struct {
			name     string
			rawURL   string
			expected string
		}{
			{name: "bare host gets // prefix", rawURL: "localhost", expected: "//localhost"},
			{name: "scheme-relative is left as is", rawURL: "//localhost", expected: "//localhost"},
			{name: "scheme already assigned", rawURL: "http://localhost", expected: "http://localhost"},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				assert.Equal(t, testCase.expected, applyDefaultScheme(testCase.rawURL, ""))
			})
		}
	})

	t.Run("with default scheme", func(t *testing.T) {
		tests := []struct {
			name     string
			rawURL   string
			expected string
		}{
			{name: "bare host gets scheme", rawURL: "localhost", expected: "http://localhost"},
			{name: "scheme already assigned", rawURL: "http://localhost", expected: "http://localhost"},
			{name: "scheme-relative gets scheme", rawURL: "//localhost", expected: "http://localhost"},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				assert.Equal(t, testCase.expected, applyDefaultScheme(testCase.rawURL, "http"))
			})
		}
	})
}

func TestCheckHost(t *testing.T) {
	// checkHost only guards against an empty host; character validation is done
	// by parseRaw via the encoding table (see TestParse for "<script>" etc).
	t.Run("non-empty host is accepted", func(t *testing.T) {
		for _, host := range []string{
			"example.com",
			"127.0.0.1",
			"[2001:db8::1]",
			"{tenant}.example.com",
		} {
			assert.NoError(t, checkHost(host), host)
		}
	})

	t.Run("empty host is rejected", func(t *testing.T) {
		assert.ErrorIs(t, checkHost(""), ErrEmptyHost)
	})
}

func TestValidatePlaceholders(t *testing.T) {
	t.Run("valid placeholders", func(t *testing.T) {
		for _, host := range []string{
			"example.com",
			"{client}.example.com",
			"api.{tenant}.example.com",
			"{region}-{client}.example.com",
			"{region}.{tenant}.host.com",
			"no-placeholders.example.com",
		} {
			assert.NoError(t, validatePlaceholders(host), host)
		}
	})

	t.Run("empty placeholder", func(t *testing.T) {
		for _, host := range []string{
			"{}.example.com",
			"{client}{}.example.com",
		} {
			assert.ErrorIs(t, validatePlaceholders(host), ErrEmptyPlaceholder, host)
		}
	})

	t.Run("unclosed placeholder", func(t *testing.T) {
		for _, host := range []string{
			"{client.example.com",
			"{client}{region.example.com",
			"{{nested}.example.com",
		} {
			assert.ErrorIs(t, validatePlaceholders(host), ErrUnclosedPlaceholder, host)
		}
	})

	t.Run("unmatched closing brace", func(t *testing.T) {
		for _, host := range []string{
			"client}.example.com",
			"{client}}.example.com",
		} {
			assert.ErrorIs(t, validatePlaceholders(host), ErrUnmatchedClosingBrace, host)
		}
	})
}
