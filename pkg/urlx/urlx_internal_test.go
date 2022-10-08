package urlx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultScheme(t *testing.T) {
	t.Run("scheme not set", func(t *testing.T) {
		tests := []struct {
			name     string
			rawURL   string
			expected string
		}{
			{
				name:     "add http scheme to url",
				rawURL:   "localhost",
				expected: "//localhost",
			},
			{
				name:     "add http scheme to url",
				rawURL:   "//localhost",
				expected: "//localhost",
			},
			{
				name:     "http scheme already assigned",
				rawURL:   "http://localhost",
				expected: "http://localhost",
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				actual := defaultScheme(testCase.rawURL, "")

				assert.Equal(t, testCase.expected, actual)
			})
		}
	})
	t.Run("scheme is set", func(t *testing.T) {
		tests := []struct {
			name     string
			rawURL   string
			scheme   string
			expected string
		}{
			{
				name:     "add http scheme to url",
				rawURL:   "localhost",
				scheme:   "http",
				expected: "http://localhost",
			},
			{
				name:     "http scheme already assigned",
				rawURL:   "http://localhost",
				scheme:   "http",
				expected: "http://localhost",
			},
			{
				name:     "set any scheme",
				rawURL:   "//localhost",
				scheme:   "http",
				expected: "http://localhost",
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				actual := defaultScheme(testCase.rawURL, "http")

				assert.Equal(t, testCase.expected, actual)
			})
		}
	})
}
