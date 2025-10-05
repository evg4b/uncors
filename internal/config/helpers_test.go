// nolint: dupl
package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/stretchr/testify/assert"
)

const (
	httpPort  = 3000
	httpsPort = 3001
)

func TestNormaliseMappings(t *testing.T) {
	t.Run("custom port handling", func(t *testing.T) {
		testsCases := []struct {
			name     string
			mappings config.Mappings
			expected config.Mappings
			useHTTPS bool
		}{
			{
				name: "correctly set http and https ports",
				mappings: config.Mappings{
					{From: hosts.Localhost.Host(), To: hosts.Github.Host()},
				},
				expected: config.Mappings{
					{From: hosts.Localhost.HTTPPort(httpPort), To: hosts.Github.Host()},
					{From: hosts.Localhost.HTTPSPort(httpsPort), To: hosts.Github.Host()},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set http port",
				mappings: config.Mappings{
					{From: hosts.Localhost.HTTP(), To: hosts.Github.HTTPS()},
				},
				expected: config.Mappings{
					{From: hosts.Localhost.HTTPPort(httpPort), To: hosts.Github.HTTPS()},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set https port",
				mappings: config.Mappings{
					{From: hosts.Localhost.HTTPS(), To: hosts.Github.HTTPS()},
				},
				expected: config.Mappings{
					{From: hosts.Localhost.HTTPSPort(httpsPort), To: hosts.Github.HTTPS()},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set mixed schemes",
				mappings: config.Mappings{
					{From: hosts.Localhost1.Host(), To: hosts.Github.HTTPS()},
					{From: hosts.Localhost2.Host(), To: hosts.Github.HTTP()},
					{From: hosts.Localhost3.HTTP(), To: hosts.APIGithub.HTTP()},
					{From: hosts.Localhost4.HTTPS(), To: hosts.APIGithub.HTTPS()},
				},
				expected: config.Mappings{
					{From: hosts.Localhost1.HTTPPort(httpPort), To: hosts.Github.HTTPS()},
					{From: hosts.Localhost1.HTTPSPort(httpsPort), To: hosts.Github.HTTPS()},
					{From: hosts.Localhost2.HTTPPort(httpPort), To: hosts.Github.HTTP()},
					{From: hosts.Localhost2.HTTPSPort(httpsPort), To: hosts.Github.HTTP()},
					{From: hosts.Localhost3.HTTPPort(httpPort), To: hosts.APIGithub.HTTP()},
					{From: hosts.Localhost4.HTTPSPort(httpsPort), To: hosts.APIGithub.HTTPS()},
				},
				useHTTPS: true,
			},
		}
		for _, testCase := range testsCases {
			t.Run(testCase.name, func(t *testing.T) {
				actual := config.NormaliseMappings(
					testCase.mappings,
					httpPort,
					httpsPort,
					testCase.useHTTPS,
				)

				assert.Equal(t, testCase.expected, actual)
			})
		}
	})

	t.Run("default port handling", func(t *testing.T) {
		httpPort, httpsPort := 80, 443
		testsCases := []struct {
			name     string
			mappings config.Mappings
			expected config.Mappings
			useHTTPS bool
		}{
			{
				name: "correctly set http and https ports",
				mappings: config.Mappings{
					{From: hosts.Localhost.Host(), To: hosts.Github.Host()},
				},
				expected: config.Mappings{
					{From: hosts.Localhost.HTTP(), To: hosts.Github.Host()},
					{From: hosts.Localhost.HTTPS(), To: hosts.Github.Host()},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set http port",
				mappings: config.Mappings{
					{From: hosts.Localhost.HTTP(), To: hosts.Github.HTTPS()},
				},
				expected: config.Mappings{
					{From: hosts.Localhost.HTTP(), To: hosts.Github.HTTPS()},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set https port",
				mappings: config.Mappings{
					{From: hosts.Localhost.HTTPS(), To: hosts.Github.HTTPS()},
				},
				expected: config.Mappings{
					{From: hosts.Localhost.HTTPS(), To: hosts.Github.HTTPS()},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set mixed schemes",
				mappings: config.Mappings{
					{From: hosts.Localhost1.Host(), To: hosts.Github.HTTPS()},
					{From: hosts.Localhost2.Host(), To: hosts.Github.HTTP()},
					{From: hosts.Localhost3.HTTP(), To: hosts.APIGithub.HTTP()},
					{From: hosts.Localhost4.HTTPS(), To: hosts.APIGithub.HTTPS()},
				},
				expected: config.Mappings{
					{From: hosts.Localhost1.HTTP(), To: hosts.Github.HTTPS()},
					{From: hosts.Localhost1.HTTPS(), To: hosts.Github.HTTPS()},
					{From: hosts.Localhost2.HTTP(), To: hosts.Github.HTTP()},
					{From: hosts.Localhost2.HTTPS(), To: hosts.Github.HTTP()},
					{From: hosts.Localhost3.HTTP(), To: hosts.APIGithub.HTTP()},
					{From: hosts.Localhost4.HTTPS(), To: hosts.APIGithub.HTTPS()},
				},
				useHTTPS: true,
			},
		}
		for _, testCase := range testsCases {
			t.Run(testCase.name, func(t *testing.T) {
				actual := config.NormaliseMappings(
					testCase.mappings,
					httpPort,
					httpsPort,
					testCase.useHTTPS,
				)

				assert.Equal(t, testCase.expected, actual)
			})
		}
	})

	t.Run("incorrect mappings", func(t *testing.T) {
		testsCases := []struct {
			name        string
			mappings    config.Mappings
			httpPort    int
			httpsPort   int
			useHTTPS    bool
			expectedErr string
		}{
			{
				name: "incorrect source url",
				mappings: config.Mappings{
					{From: "loca^host", To: hosts.Github.Host()},
				},
				httpPort:    httpPort,
				httpsPort:   httpsPort,
				useHTTPS:    true,
				expectedErr: "failed to parse source url: parse \"//loca^host\": invalid character \"^\" in host name",
			},
			{
				name: "incorrect port in source url",
				mappings: config.Mappings{
					{From: "localhost:", To: hosts.Github.Host()},
				},
				httpPort:    -1,
				httpsPort:   httpsPort,
				useHTTPS:    true,
				expectedErr: "failed to parse source url: port \"//localhost:\": empty port",
			},
		}
		for _, testCase := range testsCases {
			t.Run(testCase.name, func(t *testing.T) {
				assert.PanicsWithError(t, testCase.expectedErr, func() {
					config.NormaliseMappings(
						testCase.mappings,
						testCase.httpPort,
						testCase.httpsPort,
						testCase.useHTTPS,
					)
				})
			})
		}
	})
}
