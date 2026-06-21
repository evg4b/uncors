//go:build integration

package options_test

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
)

func TestOptionsHandlingEnabled(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://options.local"),
			To:   backend.AsHost(),
			OptionsHandling: config.OptionsHandling{
				Code:    http.StatusNoContent,
				Headers: map[string]string{"X-Options": "handled"},
			},
		}},
	})

	t.Run("preflight is answered locally with CORS and custom headers", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodOptions, env.URL("options.local", "/any/path"))
		req.Header.Set("Origin", "https://example.com")

		result := env.Do(t, req)
		defer testutils.Close(t, result.Response.Body)

		assert.Equal(t, http.StatusNoContent, result.Response.StatusCode)
		assert.Equal(t, "handled", result.Response.Header.Get("X-Options"))
		assert.Equal(t, "https://example.com", result.Response.Header.Get("Access-Control-Allow-Origin"))
		assert.False(t, result.HasBackendRequest(), "preflight must not reach the backend")

		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})
}

func TestOptionsHandlingDisabled(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From:            hosts.Parse("https://options.local"),
			To:              backend.AsHost(),
			OptionsHandling: config.OptionsHandling{Disabled: true},
		}},
	})

	t.Run("preflight is forwarded to the backend when disabled", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodOptions, env.URL("options.local", "/any/path")))
		defer testutils.Close(t, result.Response.Body)

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.True(t, result.HasBackendRequest())

		snaps.MatchSnapshot(t, result.BackendRequest(t))
		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})
}
