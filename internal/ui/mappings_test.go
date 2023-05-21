//nolint:lll
package ui_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"

	"github.com/evg4b/uncors/internal/ui"
	"github.com/stretchr/testify/assert"
)

func TestMappings(t *testing.T) {
	tests := []struct {
		name     string
		mappings []config.URLMapping
		expected string
	}{
		{
			name:     "no mapping and no mocks",
			expected: "\n",
		},
		{
			name: "http mapping only",
			mappings: []config.URLMapping{
				{From: "http://localhost", To: "https://github.com"},
			},
			expected: "PROXY: http://localhost => https://github.com\n\n",
		},
		{
			name: "http and https mappings",
			mappings: []config.URLMapping{
				{From: "http://localhost", To: "https://github.com"},
				{From: "https://localhost", To: "https://github.com"},
			},
			expected: "PROXY: https://localhost => https://github.com\nPROXY: http://localhost => https://github.com\n\n",
		},
		{
			name: "mapping and mocks",
			mappings: []config.URLMapping{
				{From: "http://localhost", To: "https://github.com", Mocks: []config.Mock{
					{}, {}, {},
				}},
				{From: "https://localhost", To: "https://github.com"},
			},
			expected: "PROXY: https://localhost => https://github.com\nPROXY: http://localhost => https://github.com\nMOCKS: 3 mock(s) registered\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ui.Mappings(tt.mappings)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
