// nolint: dupl
package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/testconstants"
	"github.com/stretchr/testify/assert"
)

func TestNormaliseMappings(t *testing.T) {
	t.Run("custom port handling", func(t *testing.T) {
		httpPort, httpsPort := 3000, 3001
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
					{From: testconstants.Host1, To: testconstants.HTTPSGithub},
					{From: "host2", To: testconstants.HTTPGithub},
					{From: "http://host3", To: "http://api.github.com"},
					{From: "https://host4", To: "https://api.github.com"},
				},
				expected: config.Mappings{
					{From: "http://host1:3000", To: testconstants.HTTPSGithub},
					{From: "https://host1:3001", To: testconstants.HTTPSGithub},
					{From: "http://host2:3000", To: testconstants.HTTPGithub},
					{From: "https://host2:3001", To: testconstants.HTTPGithub},
					{From: "http://host3:3000", To: "http://api.github.com"},
					{From: "https://host4:3001", To: "https://api.github.com"},
				},
				useHTTPS: true,
			},
		}
		for _, testCase := range testsCases {
			t.Run(testCase.name, func(t *testing.T) {
				actual, err := config.NormaliseMappings(
					testCase.mappings,
					httpPort,
					httpsPort,
					testCase.useHTTPS,
				)

				assert.NoError(t, err)
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
					{From: testconstants.Host1, To: testconstants.HTTPSGithub},
					{From: "host2", To: testconstants.HTTPGithub},
					{From: "http://host3", To: "http://api.github.com"},
					{From: "https://host4", To: "https://api.github.com"},
				},
				expected: config.Mappings{
					{From: testconstants.HTTPHost1, To: testconstants.HTTPSGithub},
					{From: testconstants.HTTPSHost1, To: testconstants.HTTPSGithub},
					{From: "http://host2", To: testconstants.HTTPGithub},
					{From: "https://host2", To: testconstants.HTTPGithub},
					{From: "http://host3", To: "http://api.github.com"},
					{From: "https://host4", To: "https://api.github.com"},
				},
				useHTTPS: true,
			},
		}
		for _, testCase := range testsCases {
			t.Run(testCase.name, func(t *testing.T) {
				actual, err := config.NormaliseMappings(
					testCase.mappings,
					httpPort,
					httpsPort,
					testCase.useHTTPS,
				)

				assert.NoError(t, err)
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
				httpPort:    3000,
				httpsPort:   3001,
				useHTTPS:    true,
				expectedErr: "failed to parse source url: parse \"//loca^host\": invalid character \"^\" in host name",
			},
			{
				name: "incorrect port in source url",
				mappings: config.Mappings{
					{From: "localhost:", To: testconstants.Github},
				},
				httpPort:    -1,
				httpsPort:   3001,
				useHTTPS:    true,
				expectedErr: "failed to parse source url: port \"//localhost:\": empty port",
			},
		}
		for _, testCase := range testsCases {
			t.Run(testCase.name, func(t *testing.T) {
				_, err := config.NormaliseMappings(
					testCase.mappings,
					testCase.httpPort,
					testCase.httpsPort,
					testCase.useHTTPS,
				)

				assert.EqualError(t, err, testCase.expectedErr)
			})
		}
	})
}
