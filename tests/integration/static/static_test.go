//go:build integration

package static_test

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
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

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "console.log('hi')", result.BodyString())
		assert.False(t, result.HasBackendRequest(), "static file must not reach the backend")

		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})

	t.Run("falls back to the index file for unknown paths under the prefix", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("static.local", "/assets/missing-route")))

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "<h1>home</h1>", result.BodyString())

		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})
}

func TestStaticAtRoot(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://static.local"),
			To:   backend.AsHost(),
			Statics: config.StaticDirectories{
				{Path: "/", Dir: "/www"},
			},
		}},
	},
		integration.WithFile("/www/app.js", "console.log('hi')"),
	)

	t.Run("serves an existing static file", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("static.local", "/app.js")))

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "console.log('hi')", result.BodyString())
		assert.False(t, result.HasBackendRequest())
	})

	for _, path := range []string{"/", "/missing.js", "/app.js/missing"} {
		t.Run("proxies "+path, func(t *testing.T) {
			result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("static.local", path)))

			assert.Equal(t, http.StatusOK, result.Response.StatusCode)
			assert.Equal(t, "ok", result.BodyString())
			assert.True(t, result.HasBackendRequest())
		})
	}
}

func TestStaticPrefixRedirect(t *testing.T) {
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
		integration.WithFile("/www/index.html", "<h1>home</h1>"),
	)

	tests := []struct {
		name     string
		path     string
		location string
	}{
		{name: "without query", path: "/assets", location: "/assets/"},
		{name: "keeps the query", path: "/assets?lang=en&v=2", location: "/assets/?lang=en&v=2"},
		{name: "keeps an escaped query", path: "/assets?q=a%26b%20c", location: "/assets/?q=a%26b%20c"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.RawURL("static.local", testCase.path)))

			assert.Equal(t, http.StatusTemporaryRedirect, result.Response.StatusCode)
			assert.Equal(t, testCase.location, result.Response.Header.Get("Location"))
			assert.False(t, result.HasBackendRequest())
		})
	}
}
