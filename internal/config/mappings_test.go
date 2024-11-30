//nolint:lll
package config_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
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
				{From: hosts.Localhost.HTTP(), To: hosts.Github.HTTPS()},
			},
			expected: []string{
				mapping(hosts.Localhost.HTTP(), hosts.Github.HTTPS()),
			},
		},
		{
			name: "http and https mappings",
			mappings: config.Mappings{
				{From: hosts.Localhost.HTTP(), To: hosts.Github.HTTPS()},
				{From: hosts.Localhost.HTTPS(), To: hosts.Github.HTTPS()},
			},
			expected: []string{
				mapping(hosts.Localhost.HTTPS(), hosts.Github.HTTPS()),
				mapping(hosts.Localhost.HTTP(), hosts.Github.HTTPS()),
			},
		},
		{
			name: "http and https mappings with statics",
			mappings: config.Mappings{
				{
					From: hosts.Localhost.HTTP(),
					To:   hosts.Github.HTTPS(),
					Statics: config.StaticDirectories{
						{
							Path: "/static",
							Dir:  "/tmp",
						},
						{
							Path:  "/static2",
							Dir:   "/tmp2",
							Index: "index.html",
						},
					},
				},
			},
			expected: []string{
				mapping(hosts.Localhost.HTTP(), hosts.Github.HTTPS()),
				"    static: /static => /tmp",
				"    static: /static2 => /tmp2",
			},
		},
		{
			name: "http and https mappings with cache",
			mappings: config.Mappings{
				{
					From: hosts.Localhost.HTTP(),
					To:   hosts.Github.HTTPS(),
					Cache: config.CacheGlobs{
						"/static",
						"/static2",
					},
				},
			},
			expected: []string{
				mapping(hosts.Localhost.HTTP(), hosts.Github.HTTPS()),
				"    cache: /static",
				"    cache: /static2",
			},
		},
		{
			name: "mapping and mocks",
			mappings: config.Mappings{
				{
					From: hosts.Localhost.HTTP(),
					To:   hosts.Github.HTTPS(),
					Mocks: config.Mocks{
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
				{From: hosts.Localhost.HTTPS(), To: hosts.Github.HTTPS()},
			},
			expected: []string{
				mapping(hosts.Localhost.HTTPS(), hosts.Github.HTTPS()),
				mapping(hosts.Localhost.HTTP(), hosts.Github.HTTPS()),
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

		assert.Equal(t, "", actual)
	})
}

func mapping(from string, to string) string {
	return fmt.Sprintf("%s => %s", from, to)
}
