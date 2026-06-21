//go:build integration

package script_test

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

func TestScriptHandler(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://script.local"),
			To:   backend.AsHost(),
			Scripts: config.Scripts{
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
			},
		}},
	})

	t.Run("inline script writes status and body without the backend", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("script.local", "/hello")))
		defer testutils.Close(t, result.Response.Body)

		body, err := io.ReadAll(result.Response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusAccepted, result.Response.StatusCode)
		assert.Equal(t, "scripted", string(body))
		assert.False(t, result.HasBackendRequest())

		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})

	t.Run("script reads path parameters from the request", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("script.local", "/echo/abc123")))
		defer testutils.Close(t, result.Response.Body)

		body, err := io.ReadAll(result.Response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "abc123", string(body))

		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})
}
