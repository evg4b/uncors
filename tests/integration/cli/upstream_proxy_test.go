//go:build integration

package cli_test

import (
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/evg4b/uncors/testing/integration"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const upstreamURL = "http://api.upstream.test"

func answering(t *testing.T, body string) http.HandlerFunc {
	t.Helper()

	return func(writer http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(writer, body)
		assert.NoError(t, err)
	}
}

func withoutProxyVariables(environ []string) []string {
	var kept []string

	for _, variable := range environ {
		name, _, _ := strings.Cut(variable, "=")
		if !strings.HasSuffix(strings.ToLower(name), "_proxy") {
			kept = append(kept, variable)
		}
	}

	return kept
}

func serveThroughUncors(t *testing.T, env []string, args ...string) string {
	t.Helper()

	port := strconv.Itoa(testutils.GetFreePort(t))
	stdout := &output{}

	cmd := integration.UncorsCommand(t, append([]string{
		"--from", "http://localhost:" + port, "--to", upstreamURL, "--interactive=false",
	}, args...))
	cmd.Env = append(withoutProxyVariables(os.Environ()), append(env, "HOME="+t.TempDir())...)
	cmd.Stdout = stdout
	cmd.Stderr = stdout
	require.NoError(t, cmd.Start())

	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	var body string

	require.Eventually(t, func() bool {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
			"http://localhost:"+port+"/ping", http.NoBody)
		if err != nil {
			return false
		}

		response, err := http.DefaultClient.Do(request)
		if err != nil {
			return false
		}

		defer response.Body.Close()

		raw, err := io.ReadAll(response.Body)
		body = string(raw)

		return err == nil && response.StatusCode == http.StatusOK
	}, outputWait, outputTick, "uncors did not answer: %s", stdout)

	return body
}

func TestUpstreamProxySettings(t *testing.T) {
	t.Run("HTTP_PROXY is used when no proxy is configured", func(t *testing.T) {
		envProxy := integration.NewBackend(t, answering(t, "env proxy"))

		body := serveThroughUncors(t, []string{"HTTP_PROXY=" + envProxy.URL()})

		assert.Equal(t, "env proxy", body)
		require.NotEmpty(t, envProxy.Requests())
		assert.True(t, strings.HasPrefix(envProxy.Requests()[0], "GET "+upstreamURL+"/ping HTTP/1.1\r\n"))
	})

	t.Run("--proxy overrides HTTP_PROXY", func(t *testing.T) {
		envProxy := integration.NewBackend(t, answering(t, "env proxy"))
		flagProxy := integration.NewBackend(t, answering(t, "flag proxy"))

		body := serveThroughUncors(t, []string{"HTTP_PROXY=" + envProxy.URL()}, "--proxy", flagProxy.URL())

		assert.Equal(t, "flag proxy", body)
		assert.Zero(t, envProxy.Count())
		require.NotEmpty(t, flagProxy.Requests())
		assert.True(t, strings.HasPrefix(flagProxy.Requests()[0], "GET "+upstreamURL+"/ping HTTP/1.1\r\n"))
	})
}
