package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/evg4b/uncors/testing/testutils/params"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

const acceptEncoding = "accept-encoding"

func TestLoadConfiguration(t *testing.T) {
	fs := testutils.PrepareFsForTests(t, "config_test_data")
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
				args: []string{params.Config, "/minimal-config.yaml"},
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
				args: []string{params.Config, "/full-config.yaml"},
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
					params.Config, "/full-config.yaml",
					params.From, mocks.SourceHost1, params.To, mocks.TargetHost1,
					params.From, mocks.SourceHost2, params.To, mocks.TargetHost2,
					params.From, mocks.SourceHost3, params.To, mocks.TargetHost3,
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
						{From: mocks.SourceHost1, To: mocks.TargetHost1},
						{From: mocks.SourceHost2, To: mocks.TargetHost2},
						{From: mocks.SourceHost3, To: mocks.TargetHost3},
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
					params.To, mocks.TargetHost1,
				},
				expected: []string{
					"recognize url mapping: `from` values are not set for every `to`",
				},
			},
			{
				name: "count of from values great then count of to",
				args: []string{
					params.From, mocks.SourceHost1, params.To, mocks.TargetHost1,
					params.From, mocks.SourceHost2,
				},
				expected: []string{
					"recognize url mapping: `to` values are not set for every `from`",
				},
			},
			{
				name: "count of to values great then count of from",
				args: []string{
					params.From, mocks.SourceHost1, params.To, mocks.TargetHost1,
					params.To, mocks.TargetHost2,
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
					"test_data/not-exist-config.yaml: no such file or directory",
				},
			},
			{
				name: "config file is corrupted",
				args: []string{
					params.Config, "/corrupted-config.yaml",
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
					params.Config, "/incorrect-config.yaml",
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
			name:     "false by default",
			config:   &config.UncorsConfig{},
			expected: false,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, testCase.config.IsHTTPSEnabled())
		})
	}
}
