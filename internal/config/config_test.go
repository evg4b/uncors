// nolint: nosprintfhostport
package config_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testconstants"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/evg4b/uncors/testing/testutils/params"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

const acceptEncoding = "accept-encoding"

const (
	corruptedConfigPath = "/corrupted-config.yaml"
	corruptedConfig     = `http-port: 8080
mappings&
  - http://demo: https://demo.com
`
)

const (
	fullConfigPath = "/full-config.yaml"
	fullConfig     = `
http-port: 8080
mappings:
  - http://localhost: https://github.com
  - from: http://localhost2
    to: https://stackoverflow.com
    mocks:
      - path: /demo
        method: POST
        queries:
          foo: bar
        headers:
          Accept-Encoding: deflate
        response:
          code: 201
          headers:
            Accept-Encoding: deflate
          raw: demo
          file: /demo.txt
proxy: localhost:8080
debug: true
https-port: 8081
cert-file: /etc/certificates/cert-file.pem
key-file: /etc/certificates/key-file.key
cache-config:
  expiration-time: 1h
  clear-time: 30m
  methods:
    - GET
    - POST
`
)

const (
	incorrectConfigPath = "/incorrect-config.yaml"
	incorrectConfig     = `http-port: xxx
mappings:
  - http://localhost: https://github.com
`
)

const (
	minimalConfigPath = "/minimal-config.yaml"
	minimalConfig     = `
http-port: 8080
mappings:
  - http://localhost: https://github.com
`
)

func TestLoadConfiguration(t *testing.T) {
	fs := testutils.FsFromMap(t, map[string]string{
		corruptedConfigPath: corruptedConfig,
		fullConfigPath:      fullConfig,
		incorrectConfigPath: incorrectConfig,
		minimalConfigPath:   minimalConfig,
	})

	t.Run("correctly parse config", func(t *testing.T) {
		tests := []struct {
			name     string
			args     []string
			expected *config.UncorsConfig
		}{
			{
				name: "return default config",
				args: []string{},
				expected: &config.UncorsConfig{
					HTTPPort:  80,
					HTTPSPort: 443,
					Mappings:  config.Mappings{},
					CacheConfig: config.CacheConfig{
						ExpirationTime: config.DefaultExpirationTime,
						ClearTime:      config.DefaultClearTime,
						Methods:        []string{http.MethodGet},
					},
				},
			},
			{
				name: "minimal config is set",
				args: []string{params.Config, minimalConfigPath},
				expected: &config.UncorsConfig{
					HTTPPort:  8080,
					HTTPSPort: 443,
					Mappings: config.Mappings{
						{From: hosts.Localhost.HTTPPort(8080), To: hosts.Github.HTTPS()},
					},
					CacheConfig: config.CacheConfig{
						ExpirationTime: config.DefaultExpirationTime,
						ClearTime:      config.DefaultClearTime,
						Methods:        []string{http.MethodGet},
					},
				},
			},
			{
				name: "read all fields from config file config is set",
				args: []string{params.Config, fullConfigPath},
				expected: &config.UncorsConfig{
					HTTPPort: 8080,
					Mappings: config.Mappings{
						{From: hosts.Localhost.HTTPPort(8080), To: hosts.Github.HTTPS()},
						{
							From: hosts.Localhost2.HTTPPort(8080),
							To:   hosts.Stackoverflow.HTTPS(),
							Mocks: config.Mocks{
								{
									Path:   "/demo",
									Method: "POST",
									Queries: map[string]string{
										"foo": "bar",
									},
									Headers: map[string]string{
										acceptEncoding: "deflate",
									},
									Response: config.Response{
										Code: 201,
										Headers: map[string]string{
											acceptEncoding: "deflate",
										},
										Raw:  "demo",
										File: "/demo.txt",
									},
								},
							},
						},
					},
					Proxy:     hosts.Localhost.Port(8080),
					Debug:     true,
					HTTPSPort: 8081,
					CertFile:  testconstants.CertFilePath,
					KeyFile:   testconstants.KeyFilePath,
					CacheConfig: config.CacheConfig{
						ExpirationTime: time.Hour,
						ClearTime:      30 * time.Minute,
						Methods: []string{
							http.MethodGet,
							http.MethodPost,
						},
					},
				},
			},
			{
				name: "read all fields from config file config is set",
				args: []string{
					params.Config, fullConfigPath,
					params.From, hosts.Localhost1.Host(), params.To, hosts.Github.Host(),
					params.From, hosts.Localhost2.Host(), params.To, hosts.Stackoverflow.Host(),
					params.From, hosts.Localhost3.Host(), params.To, hosts.APIGithub.Host(),
				},
				expected: &config.UncorsConfig{
					HTTPPort: 8080,
					Mappings: config.Mappings{
						{From: hosts.Localhost.HTTPPort(8080), To: hosts.Github.HTTPS()},
						{
							From: hosts.Localhost2.HTTPPort(8080),
							To:   hosts.Stackoverflow.HTTPS(),
							Mocks: config.Mocks{
								{
									Path:   "/demo",
									Method: "POST",
									Queries: map[string]string{
										"foo": "bar",
									},
									Headers: map[string]string{
										acceptEncoding: "deflate",
									},
									Response: config.Response{
										Code: 201,
										Headers: map[string]string{
											acceptEncoding: "deflate",
										},
										Raw:  "demo",
										File: "/demo.txt",
									},
								},
							},
						},
						{From: hosts.Localhost1.HTTPPort(8080), To: hosts.Github.Host()},
						{From: hosts.Localhost1.HTTPSPort(8081), To: hosts.Github.Host()},
						{From: hosts.Localhost2.HTTPPort(8080), To: hosts.Stackoverflow.Host()},
						{From: hosts.Localhost2.HTTPSPort(8081), To: hosts.Stackoverflow.Host()},
						{From: hosts.Localhost3.HTTPPort(8080), To: hosts.APIGithub.Host()},
						{From: hosts.Localhost3.HTTPSPort(8081), To: hosts.APIGithub.Host()},
					},
					Proxy:     hosts.Localhost.Port(8080),
					Debug:     true,
					HTTPSPort: 8081,
					CertFile:  testconstants.CertFilePath,
					KeyFile:   testconstants.KeyFilePath,
					CacheConfig: config.CacheConfig{
						ExpirationTime: time.Hour,
						ClearTime:      30 * time.Minute,
						Methods: []string{
							http.MethodGet,
							http.MethodPost,
						},
					},
				},
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				viper.Reset()
				viperInstance := viper.New()
				viperInstance.SetFs(fs)

				uncorsConfig := config.LoadConfiguration(viperInstance, testCase.args)

				assert.Equal(t, testCase.expected, uncorsConfig)
			})
		}
	})

	t.Run("parse config with error", func(t *testing.T) {
		tests := []struct {
			name     string
			args     []string
			expected []string
		}{
			{
				name: "incorrect flag provided",
				args: []string{
					"--incorrect-flag",
				},
				expected: []string{
					"filed parsing flags: unknown flag: --incorrect-flag",
				},
			},
			{
				name: "return default config",
				args: []string{
					params.To, hosts.Github.Host(),
				},
				expected: []string{
					"`from` values are not set for every `to`",
				},
			},
			{
				name: "count of from values great then count of to",
				args: []string{
					params.From, hosts.Localhost1.Host(), params.To, hosts.Github.Host(),
					params.From, hosts.Localhost2.Host(),
				},
				expected: []string{
					"`to` values are not set for every `from`",
				},
			},
			{
				name: "count of to values great then count of from",
				args: []string{
					params.From, hosts.Localhost1.Host(), params.To, hosts.Github.Host(),
					params.To, hosts.Stackoverflow.Host(),
				},
				expected: []string{
					"`from` values are not set for every `to`",
				},
			},
			{
				name: "config file doesn't exist",
				args: []string{
					params.Config, "/not-exist-config.yaml",
				},
				expected: []string{
					"filed to read config file '/not-exist-config.yaml': open /not-exist-config.yaml: file does not exist",
				},
			},
			{
				name: "config file is corrupted",
				args: []string{
					params.Config, corruptedConfigPath,
				},
				expected: []string{
					"filed to read config file '/corrupted-config.yaml': " +
						"While parsing config: yaml: line 2: could not find expected ':'",
				},
			},
			{
				name: "incorrect param type",
				args: []string{
					params.HTTPPort, "xxx",
				},
				expected: []string{
					"filed parsing flags: invalid argument \"xxx\" for \"-p, --http-port\" flag: " +
						"strconv.ParseUint: parsing \"xxx\": invalid syntax",
				},
			},
			{
				name: "incorrect type in config file",
				args: []string{
					params.Config, incorrectConfigPath,
				},
				expected: []string{
					"filed parsing config: 1 error(s) decoding:\n\n* cannot parse 'http-port' as int:" +
						" strconv.ParseInt: parsing \"xxx\": invalid syntax",
				},
			},
		}
		for _, testCase := range tests {
			testCase := testCase
			t.Run(testCase.name, func(t *testing.T) {
				for _, expected := range testCase.expected {
					viper.Reset()
					viperInstance := viper.New()
					viperInstance.SetFs(fs)

					assert.PanicsWithError(t, expected, func() {
						config.LoadConfiguration(viperInstance, testCase.args)
					})
				}
			})
		}
	})
}

func TestUncorsConfigIsHTTPSEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.UncorsConfig
		expected bool
	}{
		{
			name:     "false by default",
			config:   &config.UncorsConfig{},
			expected: false,
		},
		{
			name: "true when https configured",
			config: &config.UncorsConfig{
				HTTPSPort: 443,
				CertFile:  testconstants.CertFilePath,
				KeyFile:   testconstants.KeyFilePath,
			},
			expected: true,
		},
		{
			name: "false when https port is not configured",
			config: &config.UncorsConfig{
				CertFile: testconstants.CertFilePath,
				KeyFile:  testconstants.KeyFilePath,
			},
			expected: false,
		},
		{
			name: "false when cert file is not configured",
			config: &config.UncorsConfig{
				HTTPSPort: 443,
				KeyFile:   testconstants.KeyFilePath,
			},
			expected: false,
		},
		{
			name: "false when key file is not configured",
			config: &config.UncorsConfig{
				HTTPSPort: 443,
				CertFile:  testconstants.CertFilePath,
			},
			expected: false,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			actual := testCase.config.IsHTTPSEnabled()

			assert.Equal(t, testCase.expected, actual)
		})
	}
}
