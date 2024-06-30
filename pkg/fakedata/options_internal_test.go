package fakedata

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformOptions(t *testing.T) {
	tests := []struct {
		name     string
		options  map[string]any
		expected *gofakeit.MapParams
	}{
		{
			name:     "empty values",
			options:  map[string]any{},
			expected: &gofakeit.MapParams{},
		},
		{
			name: "string values",
			options: map[string]any{
				"demo": "demo",
			},
			expected: &gofakeit.MapParams{
				"demo": []string{"demo"},
			},
		},
		{
			name: "string array values",
			options: map[string]any{
				"demo": []string{"demo1", "demo2"},
			},
			expected: &gofakeit.MapParams{
				"demo": []string{"demo1", "demo2"},
			},
		},
		{
			name: "int values",
			options: map[string]any{
				"demo": 1,
			},
			expected: &gofakeit.MapParams{
				"demo": []string{"1"},
			},
		},
		{
			name: "float values",
			options: map[string]any{
				"demo": 1.1,
			},
			expected: &gofakeit.MapParams{
				"demo": []string{"1.1"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := transformOptions(tt.options)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}

	t.Run("invalid type", func(t *testing.T) {
		_, err := transformOptions(map[string]any{
			"demo": struct{}{},
		})

		assert.Equal(t, ErrInvalidOptionsType, err)
	})
}
