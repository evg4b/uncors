//go:build integration

package options_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/tests/integration/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// preflight sends an OPTIONS request. A non-empty origin sets the Origin header.
func preflight(t *testing.T, client *http.Client, url, origin string) *http.Response {
	t.Helper()

	request, err := http.NewRequestWithContext(context.Background(), http.MethodOptions, url, nil)
	require.NoError(t, err)

	if origin != "" {
		request.Header.Set("Origin", origin)
	}

	response, err := client.Do(request)
	require.NoError(t, err)

	return response
}

func TestOptionsHandlingEnabled(t *testing.T) {
	env := harness.New(t, harness.WithMapping(func(mapping *config.Mapping) {
		mapping.OptionsHandling = config.OptionsHandling{
			Code:    http.StatusNoContent,
			Headers: map[string]string{"X-Options": "handled"},
		}
	}))

	t.Run("preflight is answered locally with CORS and custom headers", func(t *testing.T) {
		env.Backend.Reset()

		response := preflight(t, env.Client, env.Proxy.HTTPSURL("/any/path"), "https://example.com")
		defer response.Body.Close()

		assert.Equal(t, http.StatusNoContent, response.StatusCode)
		assert.Equal(t, "handled", response.Header.Get("X-Options"))
		// CORS reflects the requesting origin for the preflight.
		assert.Equal(t, "https://example.com", response.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, 0, env.Backend.Count(), "preflight must not reach the backend")
	})
}

func TestOptionsHandlingDisabled(t *testing.T) {
	env := harness.New(t, harness.WithMapping(func(mapping *config.Mapping) {
		mapping.OptionsHandling = config.OptionsHandling{Disabled: true}
	}))

	t.Run("preflight is forwarded to the backend when disabled", func(t *testing.T) {
		env.Backend.Reset()

		response := preflight(t, env.Client, env.Proxy.HTTPSURL("/any/path"), "")
		defer response.Body.Close()

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, 1, env.Backend.Count())
	})
}
