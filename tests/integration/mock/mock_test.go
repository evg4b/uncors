//go:build integration

package mock_test

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

func do(t *testing.T, client *http.Client, method, url string) *http.Response {
	t.Helper()

	request, err := http.NewRequestWithContext(context.Background(), method, url, nil)
	require.NoError(t, err)

	response, err := client.Do(request)
	require.NoError(t, err)

	return response
}

func TestMockHandler(t *testing.T) {
	env := harness.New(t,
		harness.WithFile("/responses/data.json", `{"from":"file"}`),
		harness.WithMapping(func(mapping *config.Mapping) {
			mapping.Mocks = config.Mocks{
				{
					Matcher: config.RequestMatcher{Path: "/raw", Method: http.MethodGet},
					Response: config.Response{
						Code:    http.StatusCreated,
						Headers: map[string]string{"X-Mock": "raw"},
						Raw:     `{"mock":true}`,
					},
				},
				{
					Matcher:  config.RequestMatcher{Path: "/file"},
					Response: config.Response{Code: http.StatusOK, File: "/responses/data.json"},
				},
			}
		}),
	)

	t.Run("raw response returns configured code, headers and body", func(t *testing.T) {
		env.Backend.Reset()

		response := do(t, env.Client, http.MethodGet, env.Proxy.HTTPSURL("/raw"))
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusCreated, response.StatusCode)
		assert.Equal(t, "raw", response.Header.Get("X-Mock"))
		assert.JSONEq(t, `{"mock":true}`, string(body))
		assert.Equal(t, 0, env.Backend.Count(), "mock must not reach the backend")
	})

	t.Run("file response serves seeded file content", func(t *testing.T) {
		env.Backend.Reset()

		response := do(t, env.Client, http.MethodGet, env.Proxy.HTTPSURL("/file"))
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.JSONEq(t, `{"from":"file"}`, string(body))
		assert.Equal(t, 0, env.Backend.Count())
	})

	t.Run("method mismatch falls through to the backend", func(t *testing.T) {
		env.Backend.Reset()

		// The /raw mock only matches GET; a POST is proxied to the backend.
		response := do(t, env.Client, http.MethodPost, env.Proxy.HTTPSURL("/raw"))
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "ok", string(body))
		assert.Equal(t, 1, env.Backend.Count())
	})
}
