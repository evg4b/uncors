package configuration_test

import (
	"testing"

	"github.com/evg4b/uncors/testing/mocks"

	"github.com/evg4b/uncors/internal/configuration"
	"github.com/evg4b/uncors/internal/middlewares/mock"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfiguration(t *testing.T) {
	fs := testutils.PrepareFsForTests(t, "config_test_data")
	viperInstance := viper.New()
	viperInstance.SetFs(fs)

	t.Run("correctly parse configuration", func(t *testing.T) {
		tests := []struct {
			name     string
			args     []string
			expected *configuration.UncorsConfig
		}{
			{
				name: "return default config",
				args: []string{},
				expected: &configuration.UncorsConfig{
					HTTPPort:  80,
					HTTPSPort: 443,
					Mappings:  map[string]string{},
					Mocks:     []mock.Mock{},
				},
			},
			{
				name: "minimal config is set",
				args: []string{"--config", "/minimal-config.yaml"},
				expected: &configuration.UncorsConfig{
					HTTPPort:  8080,
					HTTPSPort: 443,
					Mappings: map[string]string{
						"http://demo": "https://demo.com",
					},
					Mocks: []mock.Mock{},
				},
			},
			{
				name: "read all fields from config file config is set",
				args: []string{"--config", "/full-config.yaml"},
				expected: &configuration.UncorsConfig{
					HTTPPort: 8080,
					Mappings: map[string]string{
						"http://demo1":       "https://demo1.com",
						"http://other-demo2": "https://demo2.io",
					},
					Proxy:     "localhost:8080",
					Debug:     true,
					HTTPSPort: 8081,
					CertFile:  "/cert-file.pem",
					KeyFile:   "/key-file.key",
					Mocks: []mock.Mock{
						{
							Path:   "/demo",
							Method: "POST",
							Queries: map[string]string{
								"foo": "bar",
							},
							Headers: map[string]string{
								"accept-encoding": "deflate",
							},
							Response: mock.Response{
								Code: 201,
								Headers: map[string]string{
									"accept-encoding": "deflate",
								},
								RawContent: "demo",
								File:       "/demo.txt",
							},
						},
					},
				},
			},
			{
				name: "read all fields from config file config is set",
				args: []string{
					"--config", "/full-config.yaml",
					"--from", mocks.SourceHost1, "--to", mocks.TargetHost1,
					"--from", mocks.SourceHost2, "--to", mocks.TargetHost2,
					"--from", mocks.SourceHost3, "--to", mocks.TargetHost3,
				},
				expected: &configuration.UncorsConfig{
					HTTPPort: 8080,
					Mappings: map[string]string{
						"http://demo1":       "https://demo1.com",
						"http://other-demo2": "https://demo2.io",
						mocks.SourceHost1:    mocks.TargetHost1,
						mocks.SourceHost2:    mocks.TargetHost2,
						mocks.SourceHost3:    mocks.TargetHost3,
					},
					Proxy:     "localhost:8080",
					Debug:     true,
					HTTPSPort: 8081,
					CertFile:  "/cert-file.pem",
					KeyFile:   "/key-file.key",
					Mocks: []mock.Mock{
						{
							Path:   "/demo",
							Method: "POST",
							Queries: map[string]string{
								"foo": "bar",
							},
							Headers: map[string]string{
								"accept-encoding": "deflate",
							},
							Response: mock.Response{
								Code: 201,
								Headers: map[string]string{
									"accept-encoding": "deflate",
								},
								RawContent: "demo",
								File:       "/demo.txt",
							},
						},
					},
				},
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				config, err := configuration.LoadConfiguration(viperInstance, testCase.args)

				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, config)
			})
		}
	})

	t.Run("parse configuration with error", func(t *testing.T) {
		tests := []struct {
			name     string
			args     []string
			expected string
		}{
			{
				name: "incorrect flag provided",
				args: []string{
					"--incorrect-flag",
				},
				expected: "filed parsing flags: unknown flag: --incorrect-flag",
			},
			{
				name: "return default config",
				args: []string{
					"--to", mocks.TargetHost1,
				},
				expected: "recognize url mapping: `from` values are not set for every `to`",
			},
			{
				name: "count of from values great then count of to",
				args: []string{
					"--from", mocks.SourceHost1, "--to", mocks.TargetHost1,
					"--from", mocks.SourceHost2,
				},
				expected: "recognize url mapping: `to` values are not set for every `from`",
			},
			{
				name: "count of to values great then count of from",
				args: []string{
					"--from", mocks.SourceHost1, "--to", mocks.TargetHost1,
					"--to", mocks.TargetHost2,
				},
				expected: "recognize url mapping: `from` values are not set for every `to`",
			},
			// TODO: Update errors for this test
			// {
			//	name: "configuration file doesn't exist",
			//	args: []string{
			//		"--config", "/not-exist-config.yaml",
			//	},
			//	expected: "filed to read config file '/not-exist-config.yaml': " +
			//		"open /Users/evg4b/Documents/uncors/internal/configuration/config_" +
			//		"test_data/not-exist-config.yaml: no such file or directory",
			// },
			{
				name: "configuration file is corrupted",
				args: []string{
					"--config", "/corrupted-config.yaml",
				},
				expected: "filed to read config file '/corrupted-config.yaml': " +
					"While parsing config: yaml: line 2: could not find expected ':'",
			},
			{
				name: "incorrect param type",
				args: []string{
					"--http-port", "xxx",
				},
				expected: "filed parsing flags: invalid argument \"xxx\" for \"--http-port\" flag: " +
					"strconv.ParseUint: parsing \"xxx\": invalid syntax",
			},
			{
				name: "incorrect type in config file",
				args: []string{
					"--config", "/incorrect-config.yaml",
				},
				expected: "filed parsing configuraion: 1 error(s) decoding:\n\n* cannot parse 'http-port' as int:" +
					" strconv.ParseInt: parsing \"xxx\": invalid syntax",
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				config, err := configuration.LoadConfiguration(viperInstance, testCase.args)

				assert.Nil(t, config)
				assert.EqualError(t, err, testCase.expected)
			})
		}
	})
}

func TestUncorsConfig_IsHTTPSEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *configuration.UncorsConfig
		expected bool
	}{
		{
			name:     "false by default",
			config:   &configuration.UncorsConfig{},
			expected: false,
		},
		{
			name:     "false by default",
			config:   &configuration.UncorsConfig{},
			expected: false,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, testCase.config.IsHTTPSEnabled())
		})
	}
}
