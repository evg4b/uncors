//go:build integration

package cli_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/evg4b/uncors/testing/integration"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const runTimeout = 10 * time.Second

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

type run struct {
	output string
	code   int
}

func runUncors(t *testing.T, args ...string) run {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), runTimeout)
	defer cancel()

	cmd := integration.UncorsCommandContext(ctx, append(args, "--interactive=false"))
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())

	raw, err := cmd.CombinedOutput()
	require.NoError(t, ctx.Err(), "uncors did not exit; it must refuse this configuration:\n%s", raw)

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "uncors must exit with an error, got %v:\n%s", err, raw)

	return run{output: ansiEscape.ReplaceAllString(string(raw), ""), code: exitErr.ExitCode()}
}

func writeFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func assertRefused(t *testing.T, result run, messages ...string) {
	t.Helper()

	assert.Equal(t, 1, result.code)
	assert.Contains(t, result.output, "ERROR")
	assert.NotContains(t, result.output, "DON'T USE IT FOR PRODUCTION", "nothing must look like a started server")
	assert.NotContains(t, result.output, "Server was stopped")

	for _, message := range messages {
		assert.Contains(t, result.output, message)
	}
}

func TestInvalidConfigFile(t *testing.T) {
	missingDir := filepath.Join(t.TempDir(), "missing")

	tests := []struct {
		name     string
		content  string
		messages []string
	}{
		{
			name:     "empty file",
			content:  "",
			messages: []string{"mappings must not be empty"},
		},
		{
			name:     "YAML syntax error",
			content:  "mappings: [ {from: x",
			messages: []string{"failed to parse config file", "yaml: line 1: did not find expected ',' or '}'"},
		},
		{
			name: "misspelled key",
			content: `mappings:
  - from: http://localhost:3000
    too: http://example.com
`,
			messages: []string{"mappings[0].too: unknown field (line 3)"},
		},
		{
			name: "duration with a space",
			content: `mappings:
  - from: http://localhost:3000
    to: http://example.com
    mocks:
      - path: /slow
        response:
          code: 200
          raw: slow
          delay: 1 s
`,
			messages: []string{
				`mappings[0].mocks[0].response.delay must be a duration without spaces, ` +
					`such as 500ms or 1m30s, got "1 s" (line 9)`,
			},
		},
		{
			name: "scalar instead of a list",
			content: `mappings:
  - from: http://localhost:3000
    to: http://example.com
    cache: 5
`,
			messages: []string{`mappings[0].cache must be a list, got "5" (line 4)`},
		},
		{
			name: "bad port in a host",
			content: `mappings:
  - from: http://localhost:abc
    to: http://example.com
`,
			messages: []string{`mappings[0].from: invalid port ":abc" after host (line 2)`},
		},
		{
			name: "every invalid mock field at once",
			content: `mappings:
  - from: http://localhost:3000
    to: http://example.com
    mocks:
      - path: users
        method: FETCH
        response:
          code: 999
          file: /no/such/file.json
`,
			messages: []string{
				"mappings[0].mocks[0].path must be absolute and start with /",
				"mappings[0].mocks[0].method must be one of GET, HEAD, POST, PUT, PATCH, DELETE, CONNECT, OPTIONS, TRACE",
				"mappings[0].mocks[0].response.code must be in range 200-599",
				"mappings[0].mocks[0].response.file /no/such/file.json does not exist",
			},
		},
		{
			name: "missing static directory",
			content: fmt.Sprintf(`mappings:
  - from: http://localhost:3000
    to: http://example.com
    statics:
      /assets: %s
`, missingDir),
			messages: []string{"mappings[0].statics[0].dir directory does not exist"},
		},
		{
			name: "script with neither code nor file",
			content: `mappings:
  - from: http://localhost:3000
    to: http://example.com
    scripts:
      - path: /api
`,
			messages: []string{"mappings[0].scripts[0].script: either 'script' or 'file' must be provided"},
		},
		{
			name: "unsupported target scheme",
			content: `mappings:
  - from: http://localhost:3000
    to: ftp://example.com
`,
			messages: []string{"mappings[0].to scheme must be http or https"},
		},
		{
			name: "duplicate source",
			content: `mappings:
  - from: http://localhost:3000
    to: http://example.com
  - from: http://localhost:3000
    to: http://example.org
`,
			messages: []string{"mappings[1].from: http://localhost:3000 is already mapped by mappings[0].from"},
		},
		{
			name: "listen address UNCORS never binds",
			content: `mappings:
  - from: http://10.1.2.3:3000
    to: http://example.com
`,
			messages: []string{"mappings[0].from: UNCORS listens on 127.0.0.1 only, so 10.1.2.3 is never reached"},
		},
		{
			name: "shared HAR file",
			content: `mappings:
  - from: http://localhost:3000
    to: http://example.com
    har: ./same.har
  - from: http://localhost:3001
    to: http://example.org
    har: ./same.har
`,
			messages: []string{`mappings[1].har: HAR file "./same.har" is already used by mappings[0].har`},
		},
		{
			name: "invalid cache settings",
			content: `cache-config:
  max-size: 0
  methods: [CONNECT]
mappings:
  - from: http://localhost:3000
    to: http://example.com
`,
			messages: []string{
				"cache-config.max-size must be greater than 0",
				"cache-config.methods[0] must be one of GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS, TRACE",
			},
		},
		{
			name: "https without a local CA",
			content: `mappings:
  - from: https://localhost:3443
    to: http://example.com
`,
			messages: []string{
				"HTTPS mapping 'localhost:3443' requires a local CA certificate for automatic TLS.",
				"uncors generate-certs",
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			path := writeFile(t, "uncors.yaml", testCase.content)

			assertRefused(t, runUncors(t, "-c", path), testCase.messages...)
		})
	}

	t.Run("missing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "absent.yaml")

		assertRefused(t, runUncors(t, "-c", path),
			fmt.Sprintf("failed to read config file '%s': open %s: no such file or directory", path, path))
	})
}

func TestInvalidFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		message string
	}{
		{name: "unknown flag", args: []string{"--wat"}, message: "failed parsing flags: unknown flag: --wat"},
		{
			name:    "from without to",
			args:    []string{"--from", "http://localhost:3000"},
			message: "`to` values are not set for every `from`",
		},
		{
			name:    "to without from",
			args:    []string{"--to", "http://example.com"},
			message: "`from` values are not set for every `to`",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			assertRefused(t, runUncors(t, testCase.args...), testCase.message)
		})
	}
}

func TestPortInUse(t *testing.T) {
	port := testutils.GetFreePort(t)

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })

	result := runUncors(t, "--from", "http://localhost:"+strconv.Itoa(port), "--to", "http://example.com")

	assert.Equal(t, 1, result.code)
	assert.Contains(t, result.output, "cannot listen on port "+strconv.Itoa(port)+": ")
	assert.Contains(t, result.output, "address already in use")
	assert.NotContains(t, result.output, "Server was stopped")
}
