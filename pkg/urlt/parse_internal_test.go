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
	t.Run("valid", func(t *testing.T) {
		for _, host := range []string{
			"example.com",
			"sub.example.com",
			"subdomain_test.example.com",
			"127.0.0.1",
			"[2001:db8::1]",
			"{tenant}.example.com",
		} {
			assert.NoError(t, checkHost(host), host)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		for _, host := range []string{"", "<script>", "ex ample.com"} {
			assert.Error(t, checkHost(host), host)
		}
	})
}
