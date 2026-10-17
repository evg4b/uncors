package config_test

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/certs"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/evg4b/uncors/testing/testutils/params"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
proxy: http://localhost:8080
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

func fullConfigExpected(proxy string) *config.UncorsConfig {
	return &config.UncorsConfig{
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
		Proxy: proxy,
		CacheConfig: config.CacheConfig{
			ExpirationTime: time.Hour,
			MaxSize:        52428800,
			Methods:        []string{http.MethodGet, http.MethodPost},
		},
		Interactive: true,
	}
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
				name:     "read all fields from config file config is set",
				args:     []string{params.Config, fullConfigPath},
				expected: fullConfigExpected("http://localhost:8080"),
			},
			{
				name: "CLI args with default ports",
				args: []string{
					params.From, hosts.Localhost1.HTTP().String(), params.To, hosts.Github.Host().String(),
					params.From, hosts.Localhost2.HTTPPort(9090).String(), params.To, hosts.Stackoverflow.Host().String(),
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
					params.From, hosts.Localhost1.HTTP().String(), params.To, hosts.Github.Host().String(),
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
				name: "CLI proxy flag overrides config file value",
				args: []string{
					params.Config, fullConfigPath,
					"--proxy", "http://newproxy:9999",
				},
				expected: fullConfigExpected("http://newproxy:9999"),
			},
			{
				name: "CLI from without scheme updates existing mapping from config file",
				args: []string{
					params.Config, minimalConfigPath,
					params.From, "LOCALHOST:8080", params.To, hosts.Stackoverflow.HTTPS().String(),
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
			{
				name: "CLI from/to updates existing mapping from config file",
				args: []string{
					params.Config, minimalConfigPath,
					params.From, hosts.Localhost.HTTPPort(8080).String(), params.To, hosts.Stackoverflow.HTTPS().String(),
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
				actual, _, err := config.LoadConfiguration(t.Context(), fs, nil, testCase.args)
				require.NoError(t, err)

				assert.Equal(t, testCase.expected, actual)
			})
		}
	})

	t.Run("returns config file path", func(t *testing.T) {
		t.Run("empty when no config file flag", func(t *testing.T) {
			args := []string{params.From, hosts.Localhost1.HTTP().String(), params.To, hosts.Github.Host().String()}
			_, configPath, err := config.LoadConfiguration(t.Context(), afero.NewMemMapFs(), nil, args)
			require.NoError(t, err)
			assert.Empty(t, configPath)
		})

		t.Run("returns the given config path", func(t *testing.T) {
			_, configPath, err := config.LoadConfiguration(t.Context(), fs, nil, []string{params.Config, minimalConfigPath})
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
				args:        []string{params.To, hosts.Github.Host().String()},
				expectedErr: "`from` values are not set for every `to`",
			},
			{
				name: "from count exceeds to count",
				args: []string{
					params.From, hosts.Localhost1.Host().String(), params.To, hosts.Github.Host().String(),
					params.From, hosts.Localhost2.Host().String(),
				},
				expectedErr: "`to` values are not set for every `from`",
			},
			{
				name: "to count exceeds from count",
				args: []string{
					params.From, hosts.Localhost1.Host().String(), params.To, hosts.Github.Host().String(),
					params.To, hosts.Stackoverflow.Host().String(),
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
				_, _, err := config.LoadConfiguration(t.Context(), fs, nil, testCase.args)
				assert.EqualError(t, err, testCase.expectedErr)
			})
		}
	})
}

func TestLoadConfiguration_VersionFlag(t *testing.T) {
	_, _, err := config.LoadConfiguration(t.Context(), afero.NewMemMapFs(), nil, []string{"--version"})
	require.ErrorIs(t, err, config.ErrVersionRequested)
}

func TestUncorsConfigValidator(t *testing.T) {
	mapFs := testutils.FsFromMap(t, map[string]string{})

	t.Run("should not register errors for", func(t *testing.T) {
		tests := []struct {
			name  string
			value *config.UncorsConfig
		}{
			{
				name: "different HAR files",
				value: &config.UncorsConfig{
					Mappings: []config.Mapping{
						{
							From: hosts.Localhost.Port(8080),
							To:   hosts.Github.HTTPS(),
							HAR:  config.HARConfig{File: "./recordings/api.har"},
						},
						{
							From: hosts.Localhost.Port(8081),
							To:   hosts.Github.HTTPS(),
							HAR:  config.HARConfig{File: "./recordings/auth.har"},
						},
						{From: hosts.Localhost.Port(8082), To: hosts.Github.HTTPS()},
						{From: hosts.Localhost.Port(8083), To: hosts.Github.HTTPS()},
					},
					CacheConfig: config.CacheConfig{
						MaxSize:        100 * 1024 * 1024,
						ExpirationTime: 10 * time.Minute,
						Methods:        []string{http.MethodGet},
					},
				},
			},
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
			{
				name: "same HAR file in several mappings",
				value: &config.UncorsConfig{
					Mappings: []config.Mapping{
						{
							From: hosts.Localhost.Port(8080),
							To:   hosts.Github.HTTPS(),
							HAR:  config.HARConfig{File: "./recordings/api.har"},
						},
						{
							From: hosts.Localhost.Port(8081),
							To:   hosts.Github.HTTPS(),
							HAR:  config.HARConfig{File: "./other.har"},
						},
						{
							From: hosts.Localhost.Port(8082),
							To:   hosts.Github.HTTPS(),
							HAR:  config.HARConfig{File: "recordings/../recordings/./api.har"},
						},
						{
							From: hosts.Localhost.Port(8083),
							To:   hosts.Github.HTTPS(),
							HAR:  config.HARConfig{File: harAbsPath(t, "other.har")},
						},
					},
					CacheConfig: config.CacheConfig{
						MaxSize:        100 * 1024 * 1024,
						ExpirationTime: 10 * time.Minute,
						Methods:        []string{http.MethodGet},
					},
				},
				error: "mappings[2].har: HAR file \"recordings/../recordings/./api.har\" is already used by mappings[0].har\n" +
					"mappings[3].har: HAR file \"" + harAbsPath(t, "other.har") + "\" is already used by mappings[1].har",
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

func harAbsPath(t *testing.T, name string) string {
	t.Helper()

	abs, err := filepath.Abs(name)
	require.NoError(t, err)

	return abs
}

func TestProxyValidatorIsValid(t *testing.T) {
	t.Run("valid url", func(t *testing.T) {
		assert.NoError(t, config.ValidateProxy("testField", "http://valid-url.com"))
	})

	t.Run("invalid url", func(t *testing.T) {
		require.EqualError(t, config.ValidateProxy("testField", "invalid:::url"), "testField is not a valid URL")
	})

	t.Run("empty url", func(t *testing.T) {
		assert.NoError(t, config.ValidateProxy("testField", ""))
	})
}

func TestUncorsConfigValidateListeners(t *testing.T) {
	fakeHome := filepath.Join(t.TempDir(), "home")
	require.NoError(t, os.MkdirAll(fakeHome, 0o755))
	t.Setenv("HOME", fakeHome)

	fs := afero.NewOsFs()
	_, _, err := certs.Generate(fs, filepath.Join(fakeHome, ".config", "uncors"), 365)
	require.NoError(t, err)

	makeConfig := func(from ...string) *config.UncorsConfig {
		mappings := make(config.Mappings, 0, len(from))
		for _, host := range from {
			mappings = append(mappings, config.Mapping{From: hosts.Parse(host), To: hosts.Github.HTTPS()})
		}

		return &config.UncorsConfig{
			Mappings: config.NormaliseMappings(mappings),
			CacheConfig: config.CacheConfig{
				MaxSize:        100 * 1024 * 1024,
				ExpirationTime: 10 * time.Minute,
				Methods:        []string{http.MethodGet},
			},
		}
	}

	t.Run("should not register errors for", func(t *testing.T) {
		tests := []struct {
			name string
			from []string
		}{
			{
				name: "different hosts on one http port",
				from: []string{"http://localhost:3000", "http://api.local:3000"},
			},
			{
				name: "different hosts on one https port",
				from: []string{"https://localhost:3443", "https://api.local:3443"},
			},
			{
				name: "same host on different ports",
				from: []string{"http://localhost:3000", "http://localhost:3001", "https://localhost:3443"},
			},
			{
				name: "http and https on default ports",
				from: []string{"http://localhost", "https://localhost"},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				require.NoError(t, makeConfig(test.from...).Validate(fs))
			})
		}
	})

	t.Run("should register errors for", func(t *testing.T) {
		tests := []struct {
			name  string
			from  []string
			error string
		}{
			{
				name:  "http and https on one port",
				from:  []string{"http://localhost:38940", "https://localhost:38940"},
				error: "mappings[1].from: port 38940 is already used for http by mappings[0].from",
			},
			{
				name:  "https on the default http port",
				from:  []string{"http://localhost", "https://api.local:80"},
				error: "mappings[1].from: port 80 is already used for http by mappings[0].from",
			},
			{
				name:  "http on the default https port",
				from:  []string{"https://localhost", "http://api.local:443"},
				error: "mappings[1].from: port 443 is already used for https by mappings[0].from",
			},
			{
				name:  "same from",
				from:  []string{"http://localhost:3000", "http://localhost:3000"},
				error: "mappings[1].from: http://localhost:3000 is already mapped by mappings[0].from",
			},
			{
				name: "same from after normalization",
				from: []string{"http://localhost", "localhost:80", "http://LOCALHOST", "https://localhost:443", "https://localhost"},
				error: "mappings[1].from: http://localhost is already mapped by mappings[0].from\n" +
					"mappings[2].from: http://LOCALHOST is already mapped by mappings[0].from\n" +
					"mappings[4].from: https://localhost is already mapped by mappings[3].from",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				require.EqualError(t, makeConfig(test.from...).Validate(fs), test.error)
			})
		}
	})
}
