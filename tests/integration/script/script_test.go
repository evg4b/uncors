//go:build integration

package script_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/tests/integration/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func get(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	require.NoError(t, err)

	response, err := client.Do(request)
	require.NoError(t, err)

	return response
}

func TestScriptHandler(t *testing.T) {
	env := harness.New(t, harness.WithMapping(func(mapping *config.Mapping) {
		mapping.Scripts = config.Scripts{
			{
				Matcher: config.RequestMatcher{Path: "/hello"},
				Script: `
response:WriteHeader(202)
response:WriteString("scripted")
`,
			},
			{
				Matcher: config.RequestMatcher{Path: "/echo/{id}"},
				Script: `
response:WriteHeader(200)
response:WriteString(request.path_params["id"])
`,
			},
		}
	}))

	t.Run("inline script writes status and body without the backend", func(t *testing.T) {
		env.Backend.Reset()

		response := get(t, env.Client, env.Proxy.HTTPSURL("/hello"))
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusAccepted, response.StatusCode)
		assert.Equal(t, "scripted", string(body))
		assert.Equal(t, 0, env.Backend.Count())
	})

	t.Run("script reads path parameters from the request", func(t *testing.T) {
		env.Backend.Reset()

		response := get(t, env.Client, env.Proxy.HTTPSURL("/echo/abc123"))
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "abc123", string(body))
	})
}
