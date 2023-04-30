//nolint:lll
package ui_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/configuration"

	"github.com/evg4b/uncors/internal/ui"
	"github.com/stretchr/testify/assert"
)

func TestMappings(t *testing.T) {
	tests := []struct {
		name      string
		mappings  []configuration.URLMapping
		mocksDefs []configuration.Mock
		expected  string
	}{
		{
			name:     "no mapping and no mocks",
			expected: "\n",
		},
		{
			name: "http mapping only",
			mappings: []configuration.URLMapping{
				{From: "http://localhost", To: "https://github.com"},
			},
			expected: "PROXY: http://localhost => https://github.com\n\n",
		},
		{
			name: "http and https mappings",
			mappings: []configuration.URLMapping{
				{From: "http://localhost", To: "https://github.com"},
				{From: "https://localhost", To: "https://github.com"},
			},
			expected: "PROXY: https://localhost => https://github.com\nPROXY: http://localhost => https://github.com\n\n",
		},
		{
			name: "one mock only",
			mocksDefs: []configuration.Mock{
				{},
			},
			expected: "MOCKS: 1 mock(s) registered\n",
		},
		{
			name: "2 mocks only",
			mocksDefs: []configuration.Mock{
				{}, {}, {},
			},
			expected: "MOCKS: 3 mock(s) registered\n",
		},
		{
			name: "mapping and mocks",
			mappings: []configuration.URLMapping{
				{From: "http://localhost", To: "https://github.com"},
				{From: "https://localhost", To: "https://github.com"},
			},
			mocksDefs: []configuration.Mock{
				{}, {}, {},
			},
			expected: "PROXY: https://localhost => https://github.com\nPROXY: http://localhost => https://github.com\nMOCKS: 3 mock(s) registered\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ui.Mappings(tt.mappings, tt.mocksDefs)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
