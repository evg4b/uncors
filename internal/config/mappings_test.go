//nolint:lll
package config_test

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/testconstants"
	"github.com/stretchr/testify/assert"
)

func TestMappings(t *testing.T) {
	tests := []struct {
		name     string
		mappings config.Mappings
		expected []string
	}{
		{
			name: "http mapping only",
			mappings: config.Mappings{
				{From: testconstants.HTTPLocalhost, To: testconstants.HTTPSGithub},
			},
			expected: []string{"http://localhost => https://github.com"},
		},
		{
			name: "http and https mappings",
			mappings: config.Mappings{
				{From: testconstants.HTTPLocalhost, To: testconstants.HTTPSGithub},
				{From: testconstants.HTTPSLocalhost, To: testconstants.HTTPSGithub},
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
					From: testconstants.HTTPLocalhost,
					To:   testconstants.HTTPSGithub,
					Mocks: []config.Mock{
						{
							Path:   "/endpoint-1",
							Method: http.MethodPost,
							Response: config.Response{
								Code: http.StatusOK,
								Raw:  "OK",
							},
						},
						{
							Path:   "/demo",
							Method: http.MethodGet,
							Queries: map[string]string{
								"param1": "value1",
							},
							Response: config.Response{
								Code: http.StatusInternalServerError,
								Raw:  "ERROR",
							},
						},
						{
							Path:   "/healthcheck",
							Method: http.MethodGet,
							Headers: map[string]string{
								"param1": "value1",
							},
							Response: config.Response{
								Code: http.StatusForbidden,
								Raw:  "ERROR",
							},
						},
					},
				},
				{From: testconstants.HTTPSLocalhost, To: testconstants.HTTPSGithub},
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

	t.Run("empty", func(t *testing.T) {
		var mappings config.Mappings

		actual := mappings.String()

		assert.Equal(t, "\n", actual)
	})
}
