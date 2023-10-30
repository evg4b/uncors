package validators

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinObjectPath(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		expected string
	}{
		{
			name:     "empty",
			paths:    []string{},
			expected: "",
		},
		{
			name:     "one",
			paths:    []string{"one"},
			expected: "one",
		},
		{
			name:     "two",
			paths:    []string{"one", "two"},
			expected: "one.two",
		},
		{
			name:     "three",
			paths:    []string{"one", "two", "three"},
			expected: "one.two.three",
		},
		{
			name:     "array",
			paths:    []string{"one", "two", "[0]"},
			expected: "one.two[0]",
		},
		{
			name:     "array with dot",
			paths:    []string{"one", "two", "[0].three"},
			expected: "one.two[0].three",
		},
		{
			name:     "array with dot and array",
			paths:    []string{"one", "two", "[0].three", "[1]"},
			expected: "one.two[0].three[1]",
		},
		{
			name:     "array with dot and array and dot",
			paths:    []string{"one", "two", "[0].three", "[1].four"},
			expected: "one.two[0].three[1].four",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := joinPath(tt.paths...)

			assert.Equal(t, tt.expected, actual)
		})
	}
}
