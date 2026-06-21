//go:build integration

package mock_test

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

func TestMockHandler(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://mock.local"),
			To:   backend.AsHost(),
			Mocks: config.Mocks{
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
			},
		}},
	}, integration.WithFile("/responses/data.json", `{"from":"file"}`))

	t.Run("raw response returns configured code, headers and body", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("mock.local", "/raw")))
		defer testutils.Close(t, result.Response.Body)

		body, err := io.ReadAll(result.Response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusCreated, result.Response.StatusCode)
		assert.Equal(t, "raw", result.Response.Header.Get("X-Mock"))
		assert.JSONEq(t, `{"mock":true}`, string(body))
		assert.False(t, result.HasBackendRequest(), "mock must not reach the backend")

		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})

	t.Run("file response serves seeded file content", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("mock.local", "/file")))
		defer testutils.Close(t, result.Response.Body)

		body, err := io.ReadAll(result.Response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.JSONEq(t, `{"from":"file"}`, string(body))
		assert.False(t, result.HasBackendRequest())

		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})

	t.Run("method mismatch falls through to the backend", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodPost, env.URL("mock.local", "/raw")))
		defer testutils.Close(t, result.Response.Body)

		body, err := io.ReadAll(result.Response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "ok", string(body))
		assert.True(t, result.HasBackendRequest())

		snaps.MatchSnapshot(t, result.BackendRequest(t))
		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})
}
