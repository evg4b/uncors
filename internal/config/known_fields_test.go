package config_test

import (
	"os"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/evg4b/uncors/testing/testutils/params"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const configFile = "/config.yaml"

func loadConfigFile(t *testing.T, fs afero.Fs, content string) (*config.UncorsConfig, error) {
	t.Helper()

	require.NoError(t, afero.WriteFile(fs, configFile, []byte(content), os.ModePerm))

	cfg, _, err := config.LoadConfiguration(t.Context(), fs, nil, []string{params.Config, configFile})

	return cfg, err
}

func TestLoadConfigurationUnknownFields(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "root",
			content: `
mapings:
  - http://localhost:3000: https://github.com
`,
			expected: "mapings: unknown field (line 2)",
		},
		{
			name: "cache-config",
			content: `
mappings:
  - http://localhost:3000: https://github.com
cache-config:
  methds: [GET]
`,
			expected: "cache-config.methds: unknown field (line 5)",
		},
		{
			name: "mapping typo in from",
			content: `
mappings:
  - frm: http://localhost:3000
`,
			expected: "mappings[0].frm: unknown field (line 3)",
		},
		{
			name: "mapping typo in single-field mapping",
			content: `
mappings:
  - form: localhost:3000
`,
			expected: "mappings[0].form: unknown field (line 3)",
		},
		{
			name: "mapping",
			content: `
mappings:
  - from: http://localhost:3000
    to: https://github.com
    mock: []
`,
			expected: "mappings[0].mock: unknown field (line 5)",
		},
		{
			name: "mock",
			content: `
mappings:
  - from: http://localhost:3000
    to: https://github.com
    mocks:
      - pth: /api
        response:
          code: 200
          raw: ok
`,
			expected: "mappings[0].mocks[0].pth: unknown field (line 6)",
		},
		{
			name: "mock response",
			content: `
mappings:
  - from: http://localhost:3000
    to: https://github.com
    mocks:
      - path: /api
        response:
          cod: 200
          raw: ok
`,
			expected: "mappings[0].mocks[0].response.cod: unknown field (line 8)",
		},
		{
			name: "script",
			content: `
mappings:
  - from: http://localhost:3000
    to: https://github.com
    scripts:
      - path: /api
        scrpt: print(1)
`,
			expected: "mappings[0].scripts[0].scrpt: unknown field (line 7)",
		},
		{
			name: "rewrite",
			content: `
mappings:
  - from: http://localhost:3000
    to: https://github.com
    rewrites:
      - form: /a
        to: /b
`,
			expected: "mappings[0].rewrites[0].form: unknown field (line 6)",
		},
		{
			name: "options-handling",
			content: `
mappings:
  - from: http://localhost:3000
    to: https://github.com
    options-handling:
      disable: true
`,
			expected: "mappings[0].options-handling.disable: unknown field (line 6)",
		},
		{
			name: "har",
			content: `
mappings:
  - from: http://localhost:3000
    to: https://github.com
    har:
      fil: ./api.har
`,
			expected: "mappings[0].har.fil: unknown field (line 6)",
		},
		{
			name: "statics list",
			content: `
mappings:
  - from: http://localhost:3000
    to: https://github.com
    statics:
      - path: /static
        dri: ./public
`,
			expected: "mappings[0].statics[0].dri: unknown field (line 7)",
		},
		{
			name: "statics map",
			content: `
mappings:
  - from: http://localhost:3000
    to: https://github.com
    statics:
      /static:
        dri: ./public
`,
			expected: "mappings[0].statics./static.dri: unknown field (line 7)",
		},
		{
			name: "merge key",
			content: `
base: &base
  to: https://github.com
  mock: []
mappings:
  - <<: *base
    from: http://localhost:3000
`,
			expected: "base: unknown field (line 2)\nmappings[0].mock: unknown field (line 4)",
		},
		{
			name: "several fields",
			content: `
mappings:
  - from: http://localhost:3000
    to: https://github.com
    mock: []
  - frm: http://localhost:4000
proxi: http://localhost:8888
`,
			expected: "mappings[0].mock: unknown field (line 5)\n" +
				"mappings[1].frm: unknown field (line 6)\n" +
				"proxi: unknown field (line 7)",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := loadConfigFile(t, afero.NewMemMapFs(), testCase.content)

			require.EqualError(t, err,
				"failed to parse config file '"+configFile+"': "+testCase.expected)
		})
	}
}

func TestLoadConfigurationEmptyFile(t *testing.T) {
	for name, content := range map[string]string{"empty": "", "comments only": "# nothing yet\n"} {
		t.Run(name, func(t *testing.T) {
			_, err := loadConfigFile(t, afero.NewMemMapFs(), content)

			require.EqualError(t, err, "mappings must not be empty")
		})
	}
}

func TestLoadConfigurationShorthand(t *testing.T) {
	t.Run("docs examples", func(t *testing.T) {
		tests := []struct {
			name     string
			content  string
			expected config.Mappings
		}{
			{
				name: "simplified syntax",
				content: `
mappings:
  - http://localhost:8080: https://github.com
`,
				expected: config.Mappings{
					{From: hosts.Localhost.HTTPPort(8080), To: hosts.Github.HTTPS()},
				},
			},
			{
				name: "mixed forms",
				content: `
mappings:
  - http://localhost:8080: https://github.com
  - http://host1:3000: https://gitlab.com
  - http://{repo}.local:8080: https://{repo}.example.io
  - from: http://host2:9090
    to: https://gitea.com
    mocks:
      - path: /api/ping
        response:
          code: 200
          raw: pong
`,
				expected: config.Mappings{
					{From: hosts.Localhost.HTTPPort(8080), To: hosts.Github.HTTPS()},
					{From: hosts.Parse("http://host1:3000"), To: hosts.Parse("https://gitlab.com")},
					{From: hosts.Parse("http://{repo}.local:8080"), To: hosts.Parse("https://{repo}.example.io")},
					{
						From: hosts.Parse("http://host2:9090"),
						To:   hosts.Parse("https://gitea.com"),
						Mocks: config.Mocks{
							{
								Matcher:  config.RequestMatcher{Path: "/api/ping"},
								Response: config.Response{Code: 200, Raw: "pong"},
							},
						},
					},
				},
			},
			{
				name: "host without scheme",
				content: `
mappings:
  - localhost: https://github.com
  - api.local: https://github.com
  - localhost:3000: https://github.com
`,
				expected: config.Mappings{
					{From: hosts.Parse("http://localhost"), To: hosts.Github.HTTPS()},
					{From: hosts.Parse("http://api.local"), To: hosts.Github.HTTPS()},
					{From: hosts.Localhost.HTTPPort(3000), To: hosts.Github.HTTPS()},
				},
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				cfg, err := loadConfigFile(t, afero.NewMemMapFs(), testCase.content)
				require.NoError(t, err)

				assert.Equal(t, testCase.expected, cfg.Mappings)
			})
		}
	})

	t.Run("full docs example", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{
			"public/index.html":   "",
			"build/docs/index.md": "",
			"mocks/world.json":    "{}",
		})

		cfg, err := loadConfigFile(t, fs, `
proxy: http://localhost:8888

cache-config:
  expiration-time: 10m
  max-size: 52428800
  methods: [GET]

mappings:
  - http://localhost:8080: https://github.com
  - from: http://other.local:3000
    to: https://example.com
    statics:
      /static: ./public
      /docs: ./build/docs
    mocks:
      - path: /hello
        response:
          code: 200
          delay: 1m30s
          raw: Hello world
      - path: /world
        method: POST
        response:
          code: 203
          delay: 5s
          file: ./mocks/world.json
    scripts:
      - path: /api/custom
        method: POST
        script: |
          response:WriteHeader(200)
    cache:
      - /api/**
    har: ./recordings/other.har
`)
		require.NoError(t, err)

		assert.Len(t, cfg.Mappings, 2)
	})
}
