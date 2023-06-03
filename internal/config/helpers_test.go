// nolint: dupl
package config_test

import (
	"testing"
	"github.com/evg4b/uncors/testing/testconstants"

	"github.com/evg4b/uncors/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNormaliseMappings(t *testing.T) {
	t.Run("custom port handling", func(t *testing.T) {
		httpPort, httpsPort := 3000, 3001
		testsCases := []struct {
			name     string
			mappings []config.Mapping
			expected []config.Mapping
			useHTTPS bool
		}{
			{
				name: "correctly set http and https ports",
				mappings: []config.Mapping{
					{From: "localhost", To: "github.com"},
				},
				expected: []config.Mapping{
					{From: "http://localhost:3000", To: "github.com"},
					{From: "https://localhost:3001", To: "github.com"},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set http port",
				mappings: []config.Mapping{
					{From: testconstants.HTTPLocalhost, To: "https://github.com"},
				},
				expected: []config.Mapping{
					{From: "http://localhost:3000", To: "https://github.com"},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set https port",
				mappings: []config.Mapping{
					{From: "https://localhost", To: "https://github.com"},
				},
				expected: []config.Mapping{
					{From: "https://localhost:3001", To: "https://github.com"},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set mixed schemes",
				mappings: []config.Mapping{
					{From: "host1", To: "https://github.com"},
					{From: "host2", To: "http://github.com"},
					{From: "http://host3", To: "http://api.github.com"},
					{From: "https://host4", To: "https://api.github.com"},
				},
				expected: []config.Mapping{
					{From: "http://host1:3000", To: "https://github.com"},
					{From: "https://host1:3001", To: "https://github.com"},
					{From: "http://host2:3000", To: "http://github.com"},
					{From: "https://host2:3001", To: "http://github.com"},
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
			mappings []config.Mapping
			expected []config.Mapping
			useHTTPS bool
		}{
			{
				name: "correctly set http and https ports",
				mappings: []config.Mapping{
					{From: "localhost", To: "github.com"},
				},
				expected: []config.Mapping{
					{From: testconstants.HTTPLocalhost, To: "github.com"},
					{From: "https://localhost", To: "github.com"},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set http port",
				mappings: []config.Mapping{
					{From: testconstants.HTTPLocalhost, To: "https://github.com"},
				},
				expected: []config.Mapping{
					{From: testconstants.HTTPLocalhost, To: "https://github.com"},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set https port",
				mappings: []config.Mapping{
					{From: "https://localhost", To: "https://github.com"},
				},
				expected: []config.Mapping{
					{From: "https://localhost", To: "https://github.com"},
				},
				useHTTPS: true,
			},
			{
				name: "correctly set mixed schemes",
				mappings: []config.Mapping{
					{From: "host1", To: "https://github.com"},
					{From: "host2", To: "http://github.com"},
					{From: "http://host3", To: "http://api.github.com"},
					{From: "https://host4", To: "https://api.github.com"},
				},
				expected: []config.Mapping{
					{From: "http://host1", To: "https://github.com"},
					{From: "https://host1", To: "https://github.com"},
					{From: "http://host2", To: "http://github.com"},
					{From: "https://host2", To: "http://github.com"},
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
			mappings    []config.Mapping
			httpPort    int
			httpsPort   int
			useHTTPS    bool
			expectedErr string
		}{
			{
				name: "incorrect source url",
				mappings: []config.Mapping{
					{From: "loca^host", To: "github.com"},
				},
				httpPort:    3000,
				httpsPort:   3001,
				useHTTPS:    true,
				expectedErr: "failed to parse source url: parse \"//loca^host\": invalid character \"^\" in host name",
			},
			{
				name: "incorrect port in source url",
				mappings: []config.Mapping{
					{From: "localhost:", To: "github.com"},
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
