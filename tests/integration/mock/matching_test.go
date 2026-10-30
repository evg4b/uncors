//go:build integration

package mock_test

import (
	"maps"
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mockDelay = 300 * time.Millisecond

func TestMockMatching(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://mock.local"),
			To:   backend.AsHost(),
			Mocks: config.Mocks{
				{
					Matcher:  config.RequestMatcher{Path: "/users/{id}"},
					Response: config.Response{Code: http.StatusOK, Raw: "any user"},
				},
				{
					Matcher:  config.RequestMatcher{Path: "/users/{id}", Method: http.MethodDelete},
					Response: config.Response{Code: http.StatusForbidden, Raw: "forbidden"},
				},
				{
					Matcher:  config.RequestMatcher{Path: "/search", Queries: map[string]string{"page": "1"}},
					Response: config.Response{Code: http.StatusOK, Raw: "first page"},
				},
				{
					Matcher: config.RequestMatcher{
						Path:    "/private",
						Headers: map[string]string{"Authorization": "Bearer token123"},
					},
					Response: config.Response{Code: http.StatusOK, Raw: "secret"},
				},
				{
					Matcher: config.RequestMatcher{Path: "/typed"},
					Response: config.Response{
						Code:    http.StatusOK,
						Headers: map[string]string{"Content-Type": "application/problem+json"},
						Raw:     `{"title":"typed"}`,
					},
				},
				{
					Matcher:  config.RequestMatcher{Path: "/slow"},
					Response: config.Response{Code: http.StatusOK, Raw: "slow", Delay: mockDelay},
				},
			},
		}},
	})

	send := func(t *testing.T, method, path string, header http.Header) *integration.Result {
		t.Helper()

		req := integration.NewRequest(t, method, env.RawURL("mock.local", path))
		maps.Copy(req.Header, header)

		return env.Do(t, req)
	}

	t.Run("a method-specific mock wins over a path-only one listed before it", func(t *testing.T) {
		deleted := send(t, http.MethodDelete, "/users/1", nil)
		fetched := send(t, http.MethodGet, "/users/1", nil)

		assert.Equal(t, http.StatusForbidden, deleted.Response.StatusCode)
		assert.Equal(t, "forbidden", deleted.BodyString())
		assert.Equal(t, http.StatusOK, fetched.Response.StatusCode)
		assert.Equal(t, "any user", fetched.BodyString())
		assert.False(t, deleted.HasBackendRequest() || fetched.HasBackendRequest())
	})

	t.Run("a query mock matches only with the listed value, ignoring extra parameters", func(t *testing.T) {
		matched := send(t, http.MethodGet, "/search?sort=asc&page=1", nil)
		other := send(t, http.MethodGet, "/search?page=2", nil)

		assert.Equal(t, "first page", matched.BodyString())
		assert.False(t, matched.HasBackendRequest())

		assert.Equal(t, "ok", other.BodyString())
		assert.Contains(t, other.BackendRequest(t), "GET /search?page=2 HTTP/1.1")
	})

	t.Run("a header mock matches only with the listed value", func(t *testing.T) {
		matched := send(t, http.MethodGet, "/private", http.Header{"Authorization": {"Bearer token123"}})
		wrong := send(t, http.MethodGet, "/private", http.Header{"Authorization": {"Bearer other"}})

		assert.Equal(t, "secret", matched.BodyString())
		assert.False(t, matched.HasBackendRequest())

		assert.Equal(t, "ok", wrong.BodyString())
		assert.Contains(t, wrong.BackendRequest(t), "Authorization: Bearer other")
	})

	t.Run("configured headers override the detected Content-Type", func(t *testing.T) {
		result := send(t, http.MethodGet, "/typed", nil)

		assert.Equal(t, "application/problem+json", result.Response.Header.Get("Content-Type"))
		assert.JSONEq(t, `{"title":"typed"}`, result.BodyString())
	})

	t.Run("a raw body without headers gets a sniffed Content-Type", func(t *testing.T) {
		result := send(t, http.MethodGet, "/users/1", nil)

		assert.Equal(t, "text/plain; charset=utf-8", result.Response.Header.Get("Content-Type"))
	})

	t.Run("delay holds the response back", func(t *testing.T) {
		started := time.Now()
		result := send(t, http.MethodGet, "/slow", nil)

		assert.GreaterOrEqual(t, time.Since(started), mockDelay)
		assert.Equal(t, "slow", result.BodyString())
	})
}

func TestMockFileRemovedAfterStart(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://mock.local"),
			To:   backend.AsHost(),
			Mocks: config.Mocks{{
				Matcher:  config.RequestMatcher{Path: "/file"},
				Response: config.Response{Code: http.StatusOK, File: "/responses/data.json"},
			}},
		}},
	}, integration.WithFile("/responses/data.json", `{}`))

	require.NoError(t, env.FS.Remove("/responses/data.json"))

	result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("mock.local", "/file")))

	assert.Equal(t, http.StatusInternalServerError, result.Response.StatusCode)
	assert.Contains(t, result.BodyString(), "An error occurred: failed to open file /responses/data.json")
	assert.False(t, result.HasBackendRequest())
}
