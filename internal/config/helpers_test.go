// nolint: dupl
package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/stretchr/testify/assert"
)

func TestNormaliseMappings(t *testing.T) {
	t.Run("port extraction and normalization", func(t *testing.T) {
		testsCases := []struct {
			name     string
			mappings config.Mappings
			expected config.Mappings
		}{
			{
				name: "custom HTTP port",
				mappings: config.Mappings{
					{From: hosts.Localhost.HTTPPort(3000), To: hosts.Github.Host()},
				},
				expected: config.Mappings{
					{From: hosts.Localhost.HTTPPort(3000), To: hosts.Github.Host()},
				},
			},
			{
				name: "custom HTTPS port",
				mappings: config.Mappings{
					{From: hosts.Localhost.HTTPSPort(3443), To: hosts.Github.HTTPS()},
				},
				expected: config.Mappings{
					{From: hosts.Localhost.HTTPSPort(3443), To: hosts.Github.HTTPS()},
				},
			},
			{
				name: "default HTTP port - should hide port in normalized URL",
				mappings: config.Mappings{
					{From: hosts.Localhost.HTTPPort(80), To: hosts.Github.HTTPS()},
				},
				expected: config.Mappings{
					{From: hosts.Localhost.HTTP(), To: hosts.Github.HTTPS()},
				},
			},
			{
				name: "default HTTPS port - should hide port in normalized URL",
				mappings: config.Mappings{
					{From: hosts.Localhost.HTTPSPort(443), To: hosts.Github.HTTPS()},
				},
				expected: config.Mappings{
					{From: hosts.Localhost.HTTPS(), To: hosts.Github.HTTPS()},
				},
			},
			{
				name: "HTTP without port - should use default 80 and hide it",
				mappings: config.Mappings{
					{From: hosts.Localhost.HTTP(), To: hosts.Github.HTTPS()},
				},
				expected: config.Mappings{
					{From: hosts.Localhost.HTTP(), To: hosts.Github.HTTPS()},
				},
			},
			{
				name: "HTTPS without port - should use default 443 and hide it",
				mappings: config.Mappings{
					{From: hosts.Localhost.HTTPS(), To: hosts.Github.HTTPS()},
				},
				expected: config.Mappings{
					{From: hosts.Localhost.HTTPS(), To: hosts.Github.HTTPS()},
				},
			},
			{
				name: "host only (no scheme, no port) - should default to HTTP with port 80",
				mappings: config.Mappings{
					{From: hosts.Localhost.Host(), To: hosts.Github.HTTPS()},
				},
				expected: config.Mappings{
					{From: hosts.Localhost.HTTP(), To: hosts.Github.HTTPS()},
				},
			},
			{
				name: "mixed ports in different mappings",
				mappings: config.Mappings{
					{From: hosts.Localhost1.HTTPPort(8080), To: hosts.Github.HTTPS()},
					{From: hosts.Localhost2.HTTPSPort(8443), To: hosts.Github.HTTP()},
					{From: hosts.Localhost3.HTTP(), To: hosts.APIGithub.HTTP()},
					{From: hosts.Localhost4.HTTPS(), To: hosts.APIGithub.HTTPS()},
				},
				expected: config.Mappings{
					{From: hosts.Localhost1.HTTPPort(8080), To: hosts.Github.HTTPS()},
					{From: hosts.Localhost2.HTTPSPort(8443), To: hosts.Github.HTTP()},
					{From: hosts.Localhost3.HTTP(), To: hosts.APIGithub.HTTP()},
					{From: hosts.Localhost4.HTTPS(), To: hosts.APIGithub.HTTPS()},
				},
			},
		}
		for _, testCase := range testsCases {
			t.Run(testCase.name, func(t *testing.T) {
				actual := config.NormaliseMappings(testCase.mappings)
				// Clear cache for comparison
				for i := range actual {
					actual[i].ClearCache()
				}
				for i := range testCase.expected {
					testCase.expected[i].ClearCache()
				}
				assert.Equal(t, testCase.expected, actual)
			})
		}
	})

	t.Run("incorrect mappings", func(t *testing.T) {
		testsCases := []struct {
			name        string
			mappings    config.Mappings
			expectedErr string
		}{
			{
				name: "incorrect source url",
				mappings: config.Mappings{
					{From: "loca^host", To: hosts.Github.Host()},
				},
				expectedErr: "failed to get host and port: parse \"//loca^host\": invalid character \"^\" in host name",
			},
			{
				name: "incorrect port in source url",
				mappings: config.Mappings{
					{From: "localhost:", To: hosts.Github.Host()},
				},
				expectedErr: "failed to get host and port: port \"//localhost:\": empty port",
			},
			{
				name: "invalid port number",
				mappings: config.Mappings{
					{From: "localhost:abc", To: hosts.Github.Host()},
				},
				expectedErr: "failed to get host and port: parse \"//localhost:abc\": invalid port \":abc\" after host",
			},
		}
		for _, testCase := range testsCases {
			t.Run(testCase.name, func(t *testing.T) {
				assert.PanicsWithError(t, testCase.expectedErr, func() {
					config.NormaliseMappings(testCase.mappings)
				})
			})
		}
	})
}
