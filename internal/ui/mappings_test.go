//nolint:lll
package ui_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/middlewares/mock"
	"github.com/evg4b/uncors/internal/ui"
	"github.com/stretchr/testify/assert"
)

func TestMappings(t *testing.T) {
	tests := []struct {
		name      string
		mappings  map[string]string
		mocksDefs []mock.Mock
		expected  string
	}{
		{
			name:     "no mapping and no mocks",
			expected: "\n",
		},
		{
			name: "http mapping only",
			mappings: map[string]string{
				"http://localhost": "https://github.com",
			},
			expected: "PROXY: http://localhost => https://github.com\n\n",
		},
		{
			name: "http and https mappings",
			mappings: map[string]string{
				"http://localhost":  "https://github.com",
				"https://localhost": "https://github.com",
			},
			expected: "PROXY: https://localhost => https://github.com\nPROXY: http://localhost => https://github.com\n\n",
		},
		{
			name: "one mock only",
			mocksDefs: []mock.Mock{
				{},
			},
			expected: "MOCKS: 1 mock(s) registered\n",
		},
		{
			name: "2 mocks only",
			mocksDefs: []mock.Mock{
				{}, {}, {},
			},
			expected: "MOCKS: 3 mock(s) registered\n",
		},
		{
			name: "mapping and mocks",
			mappings: map[string]string{
				"http://localhost":  "https://github.com",
				"https://localhost": "https://github.com",
			},
			mocksDefs: []mock.Mock{
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
