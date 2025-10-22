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
							RequestMatcher: config.RequestMatcher{
								Path:   "/endpoint-1",
								Method: http.MethodPost,
							},
							Response: config.Response{
								Code: http.StatusOK,
								Raw:  "OK",
							},
						},
						{
							RequestMatcher: config.RequestMatcher{
								Path:   "/demo",
								Method: http.MethodGet,
								Queries: map[string]string{
									"param1": "value1",
								},
							},
							Response: config.Response{
								Code: http.StatusInternalServerError,
								Raw:  "ERROR",
							},
						},
						{
							RequestMatcher: config.RequestMatcher{
								Path:   "/healthcheck",
								Method: http.MethodGet,
								Headers: map[string]string{
									"param1": "value1",
								},
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

		assert.Empty(t, actual)
	})
}

func mapping(from string, to string) string {
	return fmt.Sprintf("%s => %s", from, to)
}

func TestMappings_GroupByPort(t *testing.T) {
	tests := []struct {
		name     string
		mappings config.Mappings
		expected []config.PortGroup
	}{
		{
			name: "single HTTP mapping with default port",
			mappings: config.Mappings{
				{From: hosts.Localhost.HTTP(), To: hosts.Github.HTTPS()},
			},
			expected: []config.PortGroup{
				{
					Port:   80,
					Scheme: "http",
					Mappings: config.Mappings{
						{From: hosts.Localhost.HTTP(), To: hosts.Github.HTTPS()},
					},
				},
			},
		},
		{
			name: "single HTTPS mapping with default port",
			mappings: config.Mappings{
				{From: hosts.Localhost.HTTPS(), To: hosts.Github.HTTPS()},
			},
			expected: []config.PortGroup{
				{
					Port:   443,
					Scheme: "https",
					Mappings: config.Mappings{
						{From: hosts.Localhost.HTTPS(), To: hosts.Github.HTTPS()},
					},
				},
			},
		},
		{
			name: "HTTP and HTTPS mappings on default ports",
			mappings: config.Mappings{
				{From: hosts.Localhost.HTTP(), To: hosts.Github.HTTPS()},
				{From: hosts.Localhost.HTTPS(), To: hosts.Github.HTTPS()},
			},
			expected: []config.PortGroup{
				{
					Port:   80,
					Scheme: "http",
					Mappings: config.Mappings{
						{From: hosts.Localhost.HTTP(), To: hosts.Github.HTTPS()},
					},
				},
				{
					Port:   443,
					Scheme: "https",
					Mappings: config.Mappings{
						{From: hosts.Localhost.HTTPS(), To: hosts.Github.HTTPS()},
					},
				},
			},
		},
		{
			name: "multiple mappings on custom HTTP port",
			mappings: config.Mappings{
				{From: hosts.Localhost.HTTPPort(8080), To: hosts.Github.HTTPS()},
				{From: hosts.Localhost1.HTTPPort(8080), To: hosts.Stackoverflow.HTTPS()},
			},
			expected: []config.PortGroup{
				{
					Port:   8080,
					Scheme: "http",
					Mappings: config.Mappings{
						{From: hosts.Localhost.HTTPPort(8080), To: hosts.Github.HTTPS()},
						{From: hosts.Localhost1.HTTPPort(8080), To: hosts.Stackoverflow.HTTPS()},
					},
				},
			},
		},
		{
			name: "mappings on different ports",
			mappings: config.Mappings{
				{From: hosts.Localhost.HTTPPort(8080), To: hosts.Github.HTTPS()},
				{From: hosts.Localhost.HTTPPort(9090), To: hosts.Stackoverflow.HTTPS()},
				{From: hosts.Localhost.HTTPSPort(8443), To: hosts.Example.HTTPS()},
			},
			expected: []config.PortGroup{
				{
					Port:   8080,
					Scheme: "http",
					Mappings: config.Mappings{
						{From: hosts.Localhost.HTTPPort(8080), To: hosts.Github.HTTPS()},
					},
				},
				{
					Port:   8443,
					Scheme: "https",
					Mappings: config.Mappings{
						{From: hosts.Localhost.HTTPSPort(8443), To: hosts.Example.HTTPS()},
					},
				},
				{
					Port:   9090,
					Scheme: "http",
					Mappings: config.Mappings{
						{From: hosts.Localhost.HTTPPort(9090), To: hosts.Stackoverflow.HTTPS()},
					},
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			groups := testCase.mappings.GroupByPort()

			assert.Len(t, groups, len(testCase.expected), "number of port groups should match")

			for i, expectedGroup := range testCase.expected {
				assert.Equal(t, expectedGroup.Port, groups[i].Port, "port should match for group %d", i)
				assert.Equal(t, expectedGroup.Scheme, groups[i].Scheme, "scheme should match for group %d", i)
				assert.Len(t, groups[i].Mappings, len(expectedGroup.Mappings), "number of mappings should match for group %d", i)
			}
		})
	}

	t.Run("empty mappings", func(t *testing.T) {
		var mappings config.Mappings
		groups := mappings.GroupByPort()
		assert.Empty(t, groups)
	})

	t.Run("panic on invalid URL in GroupByPort", func(t *testing.T) {
		mappings := config.Mappings{
			{From: "://invalid-url", To: hosts.Github.HTTPS()},
		}
		assert.Panics(t, func() {
			_ = mappings.GroupByPort()
		})
	})

	t.Run("panic on invalid port in GroupByPort", func(t *testing.T) {
		mappings := config.Mappings{
			{From: "http://localhost:invalid-port", To: hosts.Github.HTTPS()},
		}
		assert.Panics(t, func() {
			_ = mappings.GroupByPort()
		})
	})
}

func TestMappings_String_Panics(t *testing.T) {
	t.Run("panic on invalid URL in String", func(t *testing.T) {
		mappings := config.Mappings{
			{From: "://invalid-url", To: hosts.Github.HTTPS()},
		}
		assert.Panics(t, func() {
			_ = mappings.String()
		})
	})
}
