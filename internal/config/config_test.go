package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
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
  - http://demo1: https://demo1.com
  - from: http://other-demo2
    to: https://demo2.io
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
cert-file: /cert-file.pem
key-file: /key-file.key
`
)

const (
	incorrectConfigPath = "/incorrect-config.yaml"
	incorrectConfig     = `http-port: xxx
mappings:
  - http://demo: https://demo.com
`
)

const (
	minimalConfigPath = "/minimal-config.yaml"
	minimalConfig     = `
http-port: 8080
mappings:
  - http://demo: https://demo.com
`
)

func TestLoadConfiguration(t *testing.T) {
	fs := testutils.FsFromMap(t, map[string]string{
		corruptedConfigPath: corruptedConfig,
		fullConfigPath:      fullConfig,
		incorrectConfigPath: incorrectConfig,
		minimalConfigPath:   minimalConfig,
	})
	viperInstance := viper.New()
	viperInstance.SetFs(fs)

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
					Mappings:  []config.Mapping{},
				},
			},
			{
				name: "minimal config is set",
				args: []string{params.Config, minimalConfigPath},
				expected: &config.UncorsConfig{
					HTTPPort:  8080,
					HTTPSPort: 443,
					Mappings: []config.Mapping{
						{From: "http://demo", To: "https://demo.com"},
					},
				},
			},
			{
				name: "read all fields from config file config is set",
				args: []string{params.Config, fullConfigPath},
				expected: &config.UncorsConfig{
					HTTPPort: 8080,
					Mappings: []config.Mapping{
						{From: "http://demo1", To: "https://demo1.com"},
						{From: "http://other-demo2", To: "https://demo2.io", Mocks: []config.Mock{
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
						}},
					},
					Proxy:     "localhost:8080",
					Debug:     true,
					HTTPSPort: 8081,
					CertFile:  "/cert-file.pem",
					KeyFile:   "/key-file.key",
				},
			},
			{
				name: "read all fields from config file config is set",
				args: []string{
					params.Config, fullConfigPath,
					params.From, testconstants.SourceHost1, params.To, testconstants.TargetHost1,
					params.From, testconstants.SourceHost2, params.To, testconstants.TargetHost2,
					params.From, testconstants.SourceHost3, params.To, testconstants.TargetHost3,
				},
				expected: &config.UncorsConfig{
					HTTPPort: 8080,
					Mappings: []config.Mapping{
						{From: "http://demo1", To: "https://demo1.com"},
						{
							From: "http://other-demo2",
							To:   "https://demo2.io",
							Mocks: []config.Mock{
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
						{From: testconstants.SourceHost1, To: testconstants.TargetHost1},
						{From: testconstants.SourceHost2, To: testconstants.TargetHost2},
						{From: testconstants.SourceHost3, To: testconstants.TargetHost3},
					},
					Proxy:     "localhost:8080",
					Debug:     true,
					HTTPSPort: 8081,
					CertFile:  "/cert-file.pem",
					KeyFile:   "/key-file.key",
				},
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				uncorsConfig, err := config.LoadConfiguration(viperInstance, testCase.args)

				assert.NoError(t, err)
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
					params.To, testconstants.TargetHost1,
				},
				expected: []string{
					"recognize url mapping: `from` values are not set for every `to`",
				},
			},
			{
				name: "count of from values great then count of to",
				args: []string{
					params.From, testconstants.SourceHost1, params.To, testconstants.TargetHost1,
					params.From, testconstants.SourceHost2,
				},
				expected: []string{
					"recognize url mapping: `to` values are not set for every `from`",
				},
			},
			{
				name: "count of to values great then count of from",
				args: []string{
					params.From, testconstants.SourceHost1, params.To, testconstants.TargetHost1,
					params.To, testconstants.TargetHost2,
				},
				expected: []string{
					"recognize url mapping: `from` values are not set for every `to`",
				},
			},
			{
				name: "config file doesn't exist",
				args: []string{
					params.Config, "/not-exist-config.yaml",
				},
				expected: []string{
					"filed to read config file '/not-exist-config.yaml': open ",
					"open /not-exist-config.yaml: file does not exist",
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
			t.Run(testCase.name, func(t *testing.T) {
				uncorsConfig, err := config.LoadConfiguration(viperInstance, testCase.args)

				assert.Nil(t, uncorsConfig)
				for _, expected := range testCase.expected {
					assert.ErrorContains(t, err, expected)
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
				CertFile:  "/cert.cer",
				KeyFile:   "/cert.key",
			},
			expected: true,
		},
		{
			name: "false when https port is not configured",
			config: &config.UncorsConfig{
				CertFile: "/cert.cer",
				KeyFile:  "/cert.key",
			},
			expected: false,
		},
		{
			name: "false when cert file is not configured",
			config: &config.UncorsConfig{
				HTTPSPort: 443,
				KeyFile:   "/cert.key",
			},
			expected: false,
		},
		{
			name: "false when key file is not configured",
			config: &config.UncorsConfig{
				HTTPSPort: 443,
				CertFile:  "/cert.cer",
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
