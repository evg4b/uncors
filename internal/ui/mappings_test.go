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
		expected []string
	}{
		{
			name:     "no mapping and no mocks",
			expected: []string{"\n"},
		},
		{
			name: "http mapping only",
			mappings: []config.URLMapping{
				{From: "http://localhost", To: "https://github.com"},
			},
			expected: []string{"PROXY: http://localhost => https://github.com"},
		},
		{
			name: "http and https mappings",
			mappings: []config.URLMapping{
				{From: "http://localhost", To: "https://github.com"},
				{From: "https://localhost", To: "https://github.com"},
			},
			expected: []string{
				"PROXY: https://localhost => https://github.com",
				"PROXY: http://localhost => https://github.com",
			},
		},
		{
			name: "mapping and mocks",
			mappings: []config.URLMapping{
				{From: "http://localhost", To: "https://github.com", Mocks: []config.Mock{
					{}, {}, {},
				}},
				{From: "https://localhost", To: "https://github.com"},
			},
			expected: []string{
				"PROXY: https://localhost => https://github.com",
				"PROXY: http://localhost => https://github.com",
				"MOCKS: 3 mock(s) registered",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ui.Mappings(tt.mappings)

			for _, expectedLine := range tt.expected {
				assert.Contains(t, actual, expectedLine)
			}
		})
	}
}
