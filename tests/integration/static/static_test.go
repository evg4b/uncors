//go:build integration

package static_test

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

func TestStaticMiddleware(t *testing.T) {
	env := harness.New(t,
		harness.WithFile("/www/app.js", "console.log('hi')"),
		harness.WithFile("/www/index.html", "<h1>home</h1>"),
		harness.WithMapping(func(mapping *config.Mapping) {
			mapping.Statics = config.StaticDirectories{
				{Path: "/assets", Dir: "/www", Index: "/index.html"},
			}
		}),
	)

	t.Run("serves an existing static file", func(t *testing.T) {
		env.Backend.Reset()

		response := get(t, env.Client, env.Proxy.HTTPSURL("/assets/app.js"))
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "console.log('hi')", string(body))
		assert.Equal(t, 0, env.Backend.Count(), "static file must not reach the backend")
	})

	t.Run("falls back to the index file for unknown paths under the prefix", func(t *testing.T) {
		env.Backend.Reset()

		response := get(t, env.Client, env.Proxy.HTTPSURL("/assets/missing-route"))
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "<h1>home</h1>", string(body))
	})
}
