//go:build integration

package rewrite_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriteRules(t *testing.T) {
	primary := integration.NewBackend(t, nil)
	auth := integration.NewBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, "auth service")
		assert.NoError(t, err)
	})
	env := integration.New(t, primary, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://rewrite.local"),
			To:   primary.AsHost(),
			Rewrites: config.RewriteOptions{
				{From: "/api/{resource}", To: "/api/v1/{resource}/list"},
				{From: "/auth/{endpoint}", To: "/v1/{endpoint}", Host: auth.AsHost()},
				{From: "/legacy", To: "/mocked"},
			},
			Mocks: config.Mocks{{
				Matcher:  config.RequestMatcher{Path: "/mocked"},
				Response: config.Response{Code: http.StatusOK, Raw: "mock"},
			}},
		}},
	})

	t.Run("keeps the query and drops the rest of the path", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.RawURL("rewrite.local", "/api/posts/42?page=2")))

		assert.Equal(t, "/api/v1/posts/list?page=2", backendPath(t, result))
	})

	t.Run("keeps the method and body", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodPost, env.URL("rewrite.local", "/api/users"))
		req.Body = io.NopCloser(strings.NewReader("payload"))
		req.ContentLength = 7

		dump := env.Do(t, req).BackendRequest(t)

		assert.True(t, strings.HasPrefix(dump, "POST /api/v1/users/list HTTP/1.1"), dump)
		assert.True(t, strings.HasSuffix(dump, "\n\npayload"), dump)
	})

	t.Run("sends a rule with host to that server only", func(t *testing.T) {
		primaryBefore := primary.Count()

		req := integration.NewRequest(t, http.MethodGet, env.URL("rewrite.local", "/auth/login"))
		req.Header.Set("Origin", "https://rewrite.local")

		result := env.Do(t, req)

		assert.Equal(t, "auth service", result.BodyString())
		assert.Equal(t, primaryBefore, primary.Count())

		requests := auth.Requests()
		require.Len(t, requests, 1)
		assert.True(t, strings.HasPrefix(requests[0], "GET /v1/login HTTP/1.1"), requests[0])
		assert.Contains(t, requests[0], "Host: "+strings.TrimPrefix(auth.URL(), "http://"))
		assert.Equal(t, "https://rewrite.local", result.Response.Header.Get("Access-Control-Allow-Origin"))
	})

	t.Run("a rewritten path is proxied even if a mock matches it", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("rewrite.local", "/legacy")))

		assert.Equal(t, "ok", result.BodyString())
		assert.Equal(t, "/mocked", backendPath(t, result))
	})

	t.Run("a path no rule matches is forwarded unchanged", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("rewrite.local", "/other/path")))

		assert.Equal(t, "/other/path", backendPath(t, result))
	})
}
