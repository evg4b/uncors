package urlreplacer_test

import (
	"testing"

	"github.com/evg4b/uncors/inernal/urlreplacer"
	"github.com/stretchr/testify/assert"
)

func TestToTarget(t *testing.T) {
	r := urlreplacer.NewSimpleReplacer("http://localhost:8080", "https://test.com")

	tests := []struct {
		name        string
		url         string
		expected    string
		expectedErr string
	}{
		{
			name:     "correctly transform root url",
			url:      "http://localhost:8080/",
			expected: "https://test.com/",
		},
		{
			name:     "correctly transform root url without slash",
			url:      "http://localhost:8080",
			expected: "https://test.com",
		},
		{
			name:     "correctly transform clear path",
			url:      "http://localhost:8080/api/info",
			expected: "https://test.com/api/info",
		},
		{
			name:     "correctly transform url with hash",
			url:      "http://localhost:8080/api/info#ancor",
			expected: "https://test.com/api/info#ancor",
		},
		{
			name:     "correctly transform url query params",
			url:      "http://localhost:8080/api/info?query=test",
			expected: "https://test.com/api/info?query=test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, _ := r.ToTarget(tt.url)

			assert.Equal(t, tt.expected, actual)
		})
	}
}
