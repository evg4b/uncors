//nolint:lll
package config_test

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestMappings(t *testing.T) {
	tests := []struct {
		name     string
		mappings config.Mappings
		expected []string
	}{
		{
			name:     "no mapping and no mocks",
			expected: []string{"\n"},
		},
		{
			name: "http mapping only",
			mappings: config.Mappings{
				{From: "http://localhost", To: "https://github.com"},
			},
			expected: []string{"http://localhost => https://github.com"},
		},
		{
			name: "http and https mappings",
			mappings: config.Mappings{
				{From: "http://localhost", To: "https://github.com"},
				{From: "https://localhost", To: "https://github.com"},
			},
			expected: []string{
				"https://localhost => https://github.com",
				"http://localhost => https://github.com",
			},
		},
		{
			name: "mapping and mocks",
			mappings: config.Mappings{
				{
					From: "http://localhost",
					To:   "https://github.com",
					Mocks: []config.Mock{
						{
							Path:   "/endpoint-1",
							Method: http.MethodPost,
							Response: config.Response{
								Code:       http.StatusOK,
								RawContent: "OK",
							},
						},
						{
							Path:   "/demo",
							Method: http.MethodGet,
							Queries: map[string]string{
								"param1": "value1",
							},
							Response: config.Response{
								Code:       http.StatusInternalServerError,
								RawContent: "ERROR",
							},
						},
						{
							Path:   "/healthcheck",
							Method: http.MethodGet,
							Headers: map[string]string{
								"param1": "value1",
							},
							Response: config.Response{
								Code:       http.StatusForbidden,
								RawContent: "ERROR",
							},
						},
					},
				},
				{From: "https://localhost", To: "https://github.com"},
			},
			expected: []string{
				"https://localhost => https://github.com",
				"http://localhost => https://github.com",
				"mock: [POST 200] /endpoint-1",
				"mock: [GET 500] /demo",
				"mock: [GET 403] /healthcheck",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.mappings.String()

			for _, expectedLine := range tt.expected {
				assert.Contains(t, actual, expectedLine)
			}
		})
	}
}
