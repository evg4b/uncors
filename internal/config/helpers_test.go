// nolint: dupl
package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/testconstants"
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
					{From: testconstants.Localhost, To: testconstants.Github},
				},
				expected: config.Mappings{
					{From: testconstants.HTTPLocalhostWithPort(httpPort), To: testconstants.Github},
					{From: testconstants.HTTPSLocalhostWithPort(httpsPort), To: testconstants.Github},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set http port",
				mappings: config.Mappings{
					{From: testconstants.HTTPLocalhost, To: testconstants.HTTPSGithub},
				},
				expected: config.Mappings{
					{From: testconstants.HTTPLocalhostWithPort(httpPort), To: testconstants.HTTPSGithub},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set https port",
				mappings: config.Mappings{
					{From: testconstants.HTTPSLocalhost, To: testconstants.HTTPSGithub},
				},
				expected: config.Mappings{
					{From: testconstants.HTTPSLocalhostWithPort(httpsPort), To: testconstants.HTTPSGithub},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set mixed schemes",
				mappings: config.Mappings{
					{From: testconstants.Localhost1, To: testconstants.HTTPSGithub},
					{From: testconstants.Localhost2, To: testconstants.HTTPGithub},
					{From: testconstants.HTTPLocalhost3, To: testconstants.HTTPAPIGithub},
					{From: testconstants.HTTPSLocalhost4, To: testconstants.HTTPSAPIGithub},
				},
				expected: config.Mappings{
					{From: testconstants.HTTPLocalhost1WithPort(httpPort), To: testconstants.HTTPSGithub},
					{From: testconstants.HTTPSLocalhost1WithPort(httpsPort), To: testconstants.HTTPSGithub},
					{From: testconstants.HTTPLocalhost2WithPort(httpPort), To: testconstants.HTTPGithub},
					{From: testconstants.HTTPSLocalhost2WithPort(httpsPort), To: testconstants.HTTPGithub},
					{From: testconstants.HTTPLocalhost3WithPort(httpPort), To: testconstants.HTTPAPIGithub},
					{From: testconstants.HTTPSLocalhost4WithPort(httpsPort), To: testconstants.HTTPSAPIGithub},
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

				assert.EqualValues(t, testCase.expected, actual)
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
					{From: testconstants.Localhost, To: testconstants.Github},
				},
				expected: config.Mappings{
					{From: testconstants.HTTPLocalhost, To: testconstants.Github},
					{From: testconstants.HTTPSLocalhost, To: testconstants.Github},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set http port",
				mappings: config.Mappings{
					{From: testconstants.HTTPLocalhost, To: testconstants.HTTPSGithub},
				},
				expected: config.Mappings{
					{From: testconstants.HTTPLocalhost, To: testconstants.HTTPSGithub},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set https port",
				mappings: config.Mappings{
					{From: testconstants.HTTPSLocalhost, To: testconstants.HTTPSGithub},
				},
				expected: config.Mappings{
					{From: testconstants.HTTPSLocalhost, To: testconstants.HTTPSGithub},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set mixed schemes",
				mappings: config.Mappings{
					{From: testconstants.Localhost1, To: testconstants.HTTPSGithub},
					{From: testconstants.Localhost2, To: testconstants.HTTPGithub},
					{From: testconstants.HTTPLocalhost3, To: testconstants.HTTPAPIGithub},
					{From: testconstants.HTTPSLocalhost4, To: testconstants.HTTPSAPIGithub},
				},
				expected: config.Mappings{
					{From: testconstants.HTTPLocalhost1, To: testconstants.HTTPSGithub},
					{From: testconstants.HTTPSLocalhost1, To: testconstants.HTTPSGithub},
					{From: testconstants.HTTPLocalhost2, To: testconstants.HTTPGithub},
					{From: testconstants.HTTPSLocalhost2, To: testconstants.HTTPGithub},
					{From: testconstants.HTTPLocalhost3, To: testconstants.HTTPAPIGithub},
					{From: testconstants.HTTPSLocalhost4, To: testconstants.HTTPSAPIGithub},
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

				assert.EqualValues(t, testCase.expected, actual)
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
					{From: "loca^host", To: testconstants.Github},
				},
				httpPort:    httpPort,
				httpsPort:   httpsPort,
				useHTTPS:    true,
				expectedErr: "failed to parse source url: parse \"//loca^host\": invalid character \"^\" in host name",
			},
			{
				name: "incorrect port in source url",
				mappings: config.Mappings{
					{From: "localhost:", To: testconstants.Github},
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
