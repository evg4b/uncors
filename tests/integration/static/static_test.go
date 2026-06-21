//go:build integration

package static_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticMiddleware(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://static.local"),
			To:   backend.AsHost(),
			Statics: config.StaticDirectories{
				{Path: "/assets", Dir: "/www", Index: "/index.html"},
			},
		}},
	},
		integration.WithFile("/www/app.js", "console.log('hi')"),
		integration.WithFile("/www/index.html", "<h1>home</h1>"),
	)

	t.Run("serves an existing static file", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("static.local", "/assets/app.js")))
		defer testutils.Close(t, result.Response.Body)

		body, err := io.ReadAll(result.Response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "console.log('hi')", string(body))
		assert.False(t, result.HasBackendRequest(), "static file must not reach the backend")

		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})

	t.Run("falls back to the index file for unknown paths under the prefix", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("static.local", "/assets/missing-route")))
		defer testutils.Close(t, result.Response.Body)

		body, err := io.ReadAll(result.Response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "<h1>home</h1>", string(body))

		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})
}
