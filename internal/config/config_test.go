package config_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/evg4b/uncors/testing/testutils/params"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clearMappingsCache clears the URL cache in all mappings for testing purposes.
func clearMappingsCache(cfg *config.UncorsConfig) {
	for i := range cfg.Mappings {
		cfg.Mappings[i].ClearCache()
	}
}

const acceptEncoding = "Accept-Encoding"

const (
	corruptedConfigPath = "/corrupted-config.yaml"
	corruptedConfig     = `mappings&
  - http://demo: https://demo.com
`
)

const (
	fullConfigPath = "/full-config.yaml"
	fullConfig     = `
mappings:
  - http://localhost:8080: https://github.com
  - from: http://localhost2:8080
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
proxy: localhost:8080
debug: true
https-port: 8081
cert-file: /etc/certificates/cert-file.pem
key-file: /etc/certificates/key-file.key
cache-config:
  expiration-time: 1h
  max-size: 52428800
  methods:
    - GET
    - POST
`
)

const (
	incorrectConfigPath = "/incorrect-config.yaml"
	incorrectConfig     = `mappings:
  - http://localhost: 123
`
)

const (
	minimalConfigPath = "/minimal-config.yaml"
	minimalConfig     = `
mappings:
  - http://localhost:8080: https://github.com
`
)

func makeTestFs(t *testing.T) afero.Fs {
	t.Helper()

	return testutils.FsFromMap(t, map[string]string{
		corruptedConfigPath: corruptedConfig,
		fullConfigPath:      fullConfig,
		incorrectConfigPath: incorrectConfig,
		minimalConfigPath:   minimalConfig,
	})
}

func TestLoadConfiguration(t *testing.T) {
	fs := makeTestFs(t)

	t.Run("correctly parse config", func(t *testing.T) {
		tests := []struct {
			name     string
			args     []string
			expected *config.UncorsConfig
		}{
			{
				name: "minimal config is set",
				args: []string{params.Config, minimalConfigPath},
				expected: &config.UncorsConfig{
					Mappings: config.Mappings{
						{From: hosts.Localhost.HTTPPort(8080), To: hosts.Github.HTTPS()},
					},
					CacheConfig: config.CacheConfig{
						ExpirationTime: config.DefaultExpirationTime,
						MaxSize:        config.DefaultMaxSize,
						Methods:        []string{http.MethodGet},
					},
					Interactive: true,
				},
			},
			{
				name: "read all fields from config file config is set",
				args: []string{params.Config, fullConfigPath},
				expected: &config.UncorsConfig{
					Mappings: config.Mappings{
						{From: hosts.Localhost.HTTPPort(8080), To: hosts.Github.HTTPS()},
						{
							From: hosts.Localhost2.HTTPPort(8080),
							To:   hosts.Stackoverflow.HTTPS(),
							Mocks: config.Mocks{
								{
									Matcher: config.RequestMatcher{
										Path:   "/demo",
										Method: "POST",
										Queries: map[string]string{
											"foo": "bar",
										},
										Headers: map[string]string{
											acceptEncoding: "deflate",
										},
									},
									Response: config.Response{
										Code: 201,
										Headers: map[string]string{
											acceptEncoding: "deflate",
										},
										Raw: "demo",
									},
								},
							},
						},
					},
					Proxy: hosts.Localhost.Port(8080),
					Debug: true,
					CacheConfig: config.CacheConfig{
						ExpirationTime: time.Hour,
						MaxSize:        52428800,
						Methods: []string{
							http.MethodGet,
							http.MethodPost,
						},
					},
					Interactive: true,
				},
			},
			{
				name: "CLI args with default ports",
				args: []string{
					params.From, hosts.Localhost1.HTTP(), params.To, hosts.Github.Host(),
					params.From, hosts.Localhost2.HTTPPort(9090), params.To, hosts.Stackoverflow.Host(),
				},
				expected: &config.UncorsConfig{
					Mappings: config.Mappings{
						{From: hosts.Localhost1.HTTP(), To: hosts.Github.Host()},
						{From: hosts.Localhost2.HTTPPort(9090), To: hosts.Stackoverflow.Host()},
					},
					CacheConfig: config.CacheConfig{
						ExpirationTime: config.DefaultExpirationTime,
						MaxSize:        config.DefaultMaxSize,
						Methods:        []string{http.MethodGet},
					},
					Interactive: true,
				},
			},
			{
				name: "interactive mode can be disabled with CLI flag",
				args: []string{
					params.From, hosts.Localhost1.HTTP(), params.To, hosts.Github.Host(),
					"--interactive=false",
				},
				expected: &config.UncorsConfig{
					Mappings: config.Mappings{
						{From: hosts.Localhost1.HTTP(), To: hosts.Github.Host()},
					},
					CacheConfig: config.CacheConfig{
						ExpirationTime: config.DefaultExpirationTime,
						MaxSize:        config.DefaultMaxSize,
						Methods:        []string{http.MethodGet},
					},
					Interactive: false,
				},
			},
			{
				name: "CLI proxy and debug flags override config file values",
				args: []string{
					params.Config, fullConfigPath,
					"--proxy", "newproxy:9999",
					"--debug=false",
				},
				expected: &config.UncorsConfig{
					Mappings: config.Mappings{
						{From: hosts.Localhost.HTTPPort(8080), To: hosts.Github.HTTPS()},
						{
							From: hosts.Localhost2.HTTPPort(8080),
							To:   hosts.Stackoverflow.HTTPS(),
							Mocks: config.Mocks{
								{
									Matcher: config.RequestMatcher{
										Path:    "/demo",
										Method:  "POST",
										Queries: map[string]string{"foo": "bar"},
										Headers: map[string]string{acceptEncoding: "deflate"},
									},
									Response: config.Response{
										Code:    201,
										Headers: map[string]string{acceptEncoding: "deflate"},
										Raw:     "demo",
									},
								},
							},
						},
					},
					Proxy: "newproxy:9999",
					Debug: false,
					CacheConfig: config.CacheConfig{
						ExpirationTime: time.Hour, MaxSize: 52428800,
						Methods: []string{http.MethodGet, http.MethodPost},
					},
					Interactive: true,
				},
			},
			{
				name: "CLI from/to updates existing mapping from config file",
				args: []string{
					params.Config, minimalConfigPath,
					params.From, hosts.Localhost.HTTPPort(8080), params.To, hosts.Stackoverflow.HTTPS(),
				},
				expected: &config.UncorsConfig{
					Mappings: config.Mappings{
						{From: hosts.Localhost.HTTPPort(8080), To: hosts.Stackoverflow.HTTPS()},
					},
					CacheConfig: config.CacheConfig{
						ExpirationTime: config.DefaultExpirationTime,
						MaxSize:        config.DefaultMaxSize,
						Methods:        []string{http.MethodGet},
					},
					Interactive: true,
				},
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				actual, _, err := config.LoadConfiguration(fs, testCase.args)
				require.NoError(t, err)

				clearMappingsCache(actual)
				clearMappingsCache(testCase.expected)

				assert.Equal(t, testCase.expected, actual)
			})
		}
	})

	t.Run("returns config file path", func(t *testing.T) {
		t.Run("empty when no config file flag", func(t *testing.T) {
			args := []string{params.From, hosts.Localhost1.HTTP(), params.To, hosts.Github.Host()}
			_, configPath, err := config.LoadConfiguration(afero.NewMemMapFs(), args)
			require.NoError(t, err)
			assert.Empty(t, configPath)
		})

		t.Run("returns the given config path", func(t *testing.T) {
			_, configPath, err := config.LoadConfiguration(fs, []string{params.Config, minimalConfigPath})
			require.NoError(t, err)
			assert.Equal(t, minimalConfigPath, configPath)
		})
	})

	t.Run("parse config with error", func(t *testing.T) {
		tests := []struct {
			name        string
			args        []string
			expectedErr string
		}{
			{
				name:        "no args produces validation error",
				args:        []string{},
				expectedErr: "mappings must not be empty",
			},
			{
				name:        "incorrect flag provided",
				args:        []string{"--incorrect-flag"},
				expectedErr: "failed parsing flags: unknown flag: --incorrect-flag",
			},
			{
				name:        "to without matching from",
				args:        []string{params.To, hosts.Github.Host()},
				expectedErr: "`from` values are not set for every `to`",
			},
			{
				name: "from count exceeds to count",
				args: []string{
					params.From, hosts.Localhost1.Host(), params.To, hosts.Github.Host(),
					params.From, hosts.Localhost2.Host(),
				},
				expectedErr: "`to` values are not set for every `from`",
			},
			{
				name: "to count exceeds from count",
				args: []string{
					params.From, hosts.Localhost1.Host(), params.To, hosts.Github.Host(),
					params.To, hosts.Stackoverflow.Host(),
				},
				expectedErr: "`from` values are not set for every `to`",
			},
			{
				name: "config file doesn't exist",
				args: []string{params.Config, "/not-exist-config.yaml"},
				expectedErr: "failed to read config file '/not-exist-config.yaml': " +
					"open /not-exist-config.yaml: file does not exist",
			},
			{
				name: "config file is corrupted",
				args: []string{params.Config, corruptedConfigPath},
				expectedErr: "failed to read config file '/corrupted-config.yaml': " +
					"While parsing config: yaml: line 2: mapping values are not allowed in this context",
			},
			{
				name: "incorrect type in config file",
				args: []string{params.Config, incorrectConfigPath},
				expectedErr: "failed to read config file '/incorrect-config.yaml': " +
					"While parsing config: mapping shorthand value must be a string URL",
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				_, _, err := config.LoadConfiguration(fs, testCase.args)
				assert.EqualError(t, err, testCase.expectedErr)
			})
		}
	})
}

func TestUncorsConfigValidator(t *testing.T) {
	mapFs := testutils.FsFromMap(t, map[string]string{})

	t.Run("should not register errors for", func(t *testing.T) {
		tests := []struct {
			name  string
			value *config.UncorsConfig
		}{
			{
				name: "minimal config",
				value: &config.UncorsConfig{
					Mappings: []config.Mapping{
						{From: hosts.Localhost.Port(8080), To: hosts.Localhost.HTTPSPort(8443)},
					},
					CacheConfig: config.CacheConfig{
						MaxSize:        100 * 1024 * 1024,
						ExpirationTime: 10 * time.Minute,
						Methods:        []string{http.MethodGet},
					},
				},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := test.value.Validate(mapFs)

				require.NoError(t, errors)
			})
		}
	})

	t.Run("should register errors for invalid config", func(t *testing.T) {
		tests := []struct {
			name  string
			value *config.UncorsConfig
			error string
		}{
			{
				name: "invalid mapping",
				value: &config.UncorsConfig{
					Mappings: []config.Mapping{},
					CacheConfig: config.CacheConfig{
						MaxSize:        100 * 1024 * 1024,
						ExpirationTime: 10 * time.Minute,
						Methods:        []string{http.MethodGet},
					},
				},
				error: "mappings must not be empty",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := test.value.Validate(mapFs)

				require.EqualError(t, errors, test.error)
			})
		}
	})
}

func TestProxyValidatorIsValid(t *testing.T) {
	t.Run("valid url", func(t *testing.T) {
		var errs config.Errors
		config.ValidateProxy("testField", "http://valid-url.com", &errs)
		assert.False(t, errs.HasAny())
	})

	t.Run("invalid url", func(t *testing.T) {
		var errs config.Errors
		config.ValidateProxy("testField", "invalid:::url", &errs)
		require.EqualError(t, errs, "testField is not a valid URL")
	})

	t.Run("empty url", func(t *testing.T) {
		var errs config.Errors
		config.ValidateProxy("testField", "", &errs)
		assert.False(t, errs.HasAny())
	})
}
