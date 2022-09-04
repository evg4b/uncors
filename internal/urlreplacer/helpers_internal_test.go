package urlreplacer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsHost(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "localhost",
			url:      "localhost",
			expected: true,
		},
		{
			name:     "host",
			url:      "demo.com",
			expected: true,
		},
		{
			name:     "host with port",
			url:      "demo.com:3000",
			expected: true,
		},
		{
			name:     "url without scheme",
			url:      "//demo.com:3000",
			expected: false,
		},
		{
			name:     "url without scheme",
			url:      "demo.com/demo/com",
			expected: false,
		},
		{
			name:     "url without scheme",
			url:      "demo.com?demo=com",
			expected: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := isHost(testCase.url)

			assert.Equal(t, testCase.expected, actual)
		})
	}
}
