//go:build integration

package static_test

import (
	"maps"
	"net/http"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/stretchr/testify/assert"
)

func TestStaticRequestHandling(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://static.local"),
			To:   backend.AsHost(),
			Statics: config.StaticDirectories{
				{Path: "/assets", Dir: "/www"},
				{Path: "/app", Dir: "/spa", Index: "index.html"},
			},
			Mocks: config.Mocks{{
				Matcher:  config.RequestMatcher{Path: "/assets/mocked.js"},
				Response: config.Response{Code: http.StatusOK, Raw: "mock"},
			}},
		}},
	},
		integration.WithFile("/www/site.css", "body{}"),
		integration.WithFile("/www/digits.txt", "0123456789"),
		integration.WithFile("/www/nested/page.txt", "nested"),
		integration.WithFile("/spa/index.html", "<h1>spa</h1>"),
		integration.WithFile("/spa/users/readme.txt", "readme"),
	)

	request := func(t *testing.T, method, path string, header http.Header) *integration.Result {
		t.Helper()

		req := integration.NewRequest(t, method, env.URL("static.local", path))
		maps.Copy(req.Header, header)

		return env.Do(t, req)
	}

	t.Run("serves a file with a Content-Type from its extension and no CORS headers", func(t *testing.T) {
		result := request(t, http.MethodGet, "/assets/site.css", http.Header{"Origin": {"https://front.local"}})

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "body{}", result.BodyString())
		assert.Equal(t, "text/css; charset=utf-8", result.Response.Header.Get("Content-Type"))
		assert.Empty(t, corsHeaders(result.Response.Header))
		assert.False(t, result.HasBackendRequest())
	})

	t.Run("answers HEAD with the headers only", func(t *testing.T) {
		result := request(t, http.MethodHead, "/assets/site.css", nil)

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, int64(len("body{}")), result.Response.ContentLength)
		assert.Empty(t, result.BodyString())
		assert.False(t, result.HasBackendRequest())
	})

	t.Run("serves a byte range", func(t *testing.T) {
		result := request(t, http.MethodGet, "/assets/digits.txt", http.Header{"Range": {"bytes=2-4"}})

		assert.Equal(t, http.StatusPartialContent, result.Response.StatusCode)
		assert.Equal(t, "234", result.BodyString())
		assert.Equal(t, "bytes 2-4/10", result.Response.Header.Get("Content-Range"))
	})

	t.Run("answers a conditional request with 304", func(t *testing.T) {
		first := request(t, http.MethodGet, "/assets/site.css", nil)
		modified := first.Response.Header.Get("Last-Modified")

		result := request(t, http.MethodGet, "/assets/site.css", http.Header{"If-Modified-Since": {modified}})

		assert.Equal(t, http.StatusNotModified, result.Response.StatusCode)
		assert.Empty(t, result.BodyString())
	})

	t.Run("serves a nested file", func(t *testing.T) {
		result := request(t, http.MethodGet, "/assets/nested/page.txt", nil)

		assert.Equal(t, "nested", result.BodyString())
	})

	t.Run("sends other methods under the prefix to the backend", func(t *testing.T) {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
			result := request(t, method, "/assets/site.css", nil)

			assert.Equal(t, "ok", result.BodyString(), method)
			assert.True(t, strings.HasPrefix(result.BackendRequest(t), method+" /assets/site.css HTTP/1.1"), method)
		}
	})

	t.Run("a directory without index goes to the backend", func(t *testing.T) {
		result := request(t, http.MethodGet, "/assets/nested/", nil)

		assert.Equal(t, "ok", result.BodyString())
		assert.Contains(t, result.BackendRequest(t), "GET /assets/nested/ HTTP/1.1")
	})

	t.Run("a directory with index set returns the index", func(t *testing.T) {
		result := request(t, http.MethodGet, "/app/users/", nil)

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "<h1>spa</h1>", result.BodyString())
		assert.False(t, result.HasBackendRequest())
	})

	t.Run("a mock under a static prefix is never reached", func(t *testing.T) {
		result := request(t, http.MethodGet, "/assets/mocked.js", nil)

		assert.Equal(t, "ok", result.BodyString())
		assert.True(t, result.HasBackendRequest())
	})
}

func corsHeaders(header http.Header) []string {
	var names []string

	for name := range header {
		if strings.HasPrefix(name, "Access-Control-") {
			names = append(names, name)
		}
	}

	return names
}
