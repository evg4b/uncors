package config_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var localhostSecure = "https://localhost:9090"

func TestMappingUnmarshalYAML(t *testing.T) {
	t.Run("positive cases", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected config.Mapping
		}{
			{
				name:  "simple key-value shorthand",
				input: "http://localhost:4200: https://github.com",
				expected: config.Mapping{
					From: hosts.Localhost.HTTPPort(4200),
					To:   hosts.Github.HTTPS(),
				},
			},
			{
				name:  "full object mapping",
				input: "{ from: http://localhost:3000, to: https://api.github.com }",
				expected: config.Mapping{
					From: hosts.Localhost.HTTPPort(3000),
					To:   hosts.APIGithub.HTTPS(),
				},
			},
			{
				name: "mapping with HAR shorthand",
				input: `
from: http://localhost:3000
to: https://api.example.com
har: ./recordings/api.har
`,
				expected: config.Mapping{
					From: hosts.Localhost.HTTPPort(3000),
					To:   "https://api.example.com",
					HAR:  config.HARConfig{File: "./recordings/api.har"},
				},
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				var actual config.Mapping
				require.NoError(t, yaml.Unmarshal([]byte(testCase.input), &actual))
				assert.Equal(t, testCase.expected, actual)
			})
		}
	})

	t.Run("error cases", func(t *testing.T) {
		t.Run("shorthand with non-string value", func(t *testing.T) {
			var actual config.Mapping

			err := yaml.Unmarshal([]byte("http://localhost: 123"), &actual)
			assert.Error(t, err)
		})
	})
}

func TestURLMappingClone(t *testing.T) {
	tests := []struct {
		name     string
		expected config.Mapping
	}{
		{
			name:     "empty structure",
			expected: config.Mapping{},
		},
		{
			name: "structure with 1 field",
			expected: config.Mapping{
				From: hosts.Localhost.HTTP(),
			},
		},
		{
			name: "structure with 2 field",
			expected: config.Mapping{
				From: hosts.Localhost.HTTP(),
				To:   localhostSecure,
			},
		},
		{
			name: "structure with inner collections",
			expected: config.Mapping{
				From: hosts.Localhost.HTTP(),
				To:   localhostSecure,
				Statics: []config.StaticDirectory{
					{Path: "/cc", Dir: "cc"},
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			actual := testCase.expected.Clone()

			assert.NotSame(t, &testCase.expected, &actual)
			assert.Equal(t, testCase.expected, actual)
			assert.NotSame(t, &testCase.expected.Statics, &actual.Statics)
		})
	}
}

func TestMappingValidator(t *testing.T) {
	const (
		field        = "mapping"
		demoJSONPath = "/tmp/demo.json"
	)

	t.Run("should not register errors for", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{
			demoJSONPath: "{}",
		})

		tests := []struct {
			name  string
			value config.Mapping
		}{
			{
				name: "full filled mapping",
				value: config.Mapping{
					From: "localhost",
					To:   hosts.Github.Host(),
					Statics: []config.StaticDirectory{
						{Path: "/", Dir: "/tmp"},
						{Path: "/", Dir: "/tmp"},
					},
					Mocks: []config.Mock{
						{
							Matcher: config.RequestMatcher{
								Path:   "/api/info",
								Method: http.MethodGet,
							},
							Response: config.Response{
								Code: 200,
								Raw:  "test",
							},
						},
						{
							Matcher: config.RequestMatcher{
								Path:   "/api/info/demo",
								Method: http.MethodGet,
							},
							Response: config.Response{
								Code: 300,
								File: demoJSONPath,
							},
						},
					},
					Cache: config.CacheGlobs{
						"/api/constants",
						"/**",
					},
				},
			},
			{
				name: "mapping without mocks and statics and caches",
				value: config.Mapping{
					From:    "localhost",
					To:      hosts.Github.Host(),
					Statics: []config.StaticDirectory{},
					Mocks:   []config.Mock{},
					Cache:   config.CacheGlobs{},
				},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				assert.NoError(t, test.value.Validate(field, fs))
			})
		}
	})

	t.Run("should register errors for", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{
			demoJSONPath: "{}",
		})

		tests := []struct {
			name  string
			value config.Mapping
			error string
		}{
			{
				name: "mapping without from",
				value: config.Mapping{
					From:    "",
					To:      hosts.Github.Host(),
					Statics: []config.StaticDirectory{},
					Mocks:   []config.Mock{},
					Cache:   config.CacheGlobs{},
				},
				error: "mapping.from must not be empty",
			},
			{
				name: "mapping without to",
				value: config.Mapping{
					From:    "localhost",
					To:      "",
					Statics: []config.StaticDirectory{},
					Mocks:   []config.Mock{},
					Cache:   config.CacheGlobs{},
				},
				error: "mapping.to must not be empty",
			},
			{
				name: "mapping with invalid statics",
				value: config.Mapping{
					From: "localhost",
					To:   hosts.Github.Host(),
					Statics: []config.StaticDirectory{
						{Path: "/", Dir: "/tmp"},
						{Path: "/", Dir: ""},
					},
					Mocks: []config.Mock{},
					Cache: config.CacheGlobs{},
				},
				error: "mapping.statics[1].directory must not be empty",
			},
			{
				name: "mapping with invalid mocks",
				value: config.Mapping{
					From:    "localhost",
					To:      hosts.Github.Host(),
					Statics: []config.StaticDirectory{},
					Mocks: []config.Mock{
						{
							Matcher: config.RequestMatcher{
								Path:   "/api/user",
								Method: "invalid",
							},
							Response: config.Response{
								Code: 200,
								Raw:  "test",
							},
						},
					},
					Cache: config.CacheGlobs{},
				},
				error: "mapping.mocks[0].method must be one of GET, HEAD, POST, PUT, PATCH, DELETE, CONNECT, OPTIONS, TRACE",
			},
			{
				name: "mapping with invalid cache glob",
				value: config.Mapping{
					From:    "localhost",
					To:      hosts.Github.Host(),
					Statics: []config.StaticDirectory{},
					Mocks:   []config.Mock{},
					Cache: config.CacheGlobs{
						"/api/info[",
					},
				},
				error: "mapping.cache[0] is not a valid glob pattern",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				require.EqualError(t, test.value.Validate(field, fs), test.error)
			})
		}
	})
}

func TestValidateTLS(t *testing.T) {
	t.Run("skip validation for invalid URL", func(t *testing.T) {
		err := config.ValidateTLS(
			"test",
			config.Mapping{From: "://invalid-url", To: hosts.Example.HTTP()},
			afero.NewMemMapFs(),
		)
		assert.NoError(t, err)
	})

	t.Run("skip validation for non-HTTPS", func(t *testing.T) {
		err := config.ValidateTLS(
			"test",
			config.Mapping{From: "http://localhost:8080", To: hosts.Example.HTTP()},
			afero.NewMemMapFs(),
		)
		assert.NoError(t, err)
	})

	t.Run("error when CA does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		err := config.ValidateTLS("test",
			config.Mapping{From: "https://localhost:8443", To: hosts.Example.HTTP()},
			afero.NewOsFs())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "HTTPS mapping 'localhost:8443' requires a local CA certificate")
		assert.Contains(t, err.Error(), "uncors generate-certs")
	})

	t.Run("pass when CA exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		caDir := filepath.Join(fakeHome, ".config", "uncors")
		_, _, err := infratls.GenerateCA(infratls.CAConfig{ValidityDays: 365, OutputDir: caDir, Fs: fs})
		require.NoError(t, err)

		err = config.ValidateTLS("test",
			config.Mapping{From: "https://localhost:8443", To: hosts.Example.HTTP()},
			fs)

		assert.NoError(t, err)
	})
}

func TestValidateGlobPatternForCache(t *testing.T) {
	const field = "cache"

	t.Run("should not register errors for", func(t *testing.T) {
		patterns := []string{"/api/**", "/constants", "/translations", "/**/*.js", "/**", "/[12]/demo", "**", "*"}
		for _, pattern := range patterns {
			p := pattern
			t.Run(fmt.Sprintf("%s pattern", p), func(t *testing.T) {
				assert.NoError(t, config.ValidateGlobPattern(field, p))
			})
		}
	})

	t.Run("should register errors for", func(t *testing.T) {
		tests := []struct{ pattern, error string }{
			{pattern: "/[12/demo", error: "cache is not a valid glob pattern"},
			{pattern: "/{{12}/demo", error: "cache is not a valid glob pattern"},
		}
		for _, test := range tests {
			t.Run(fmt.Sprintf("%s test", test.pattern), func(t *testing.T) {
				require.EqualError(t, config.ValidateGlobPattern(field, test.pattern), test.error)
			})
		}
	})
}
