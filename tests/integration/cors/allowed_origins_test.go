//go:build integration

package cors_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

const (
	allowedOrigin  = "http://localhost:3000"
	rejectedOrigin = "http://attacker.test"
)

func corsHeaderNames(header http.Header) []string {
	var names []string

	for name := range header {
		if strings.HasPrefix(name, "Access-Control-") {
			names = append(names, name)
		}
	}

	return names
}

func TestAllowedOrigins(t *testing.T) {
	backend := integration.NewBackend(t, func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set(headers.AccessControlAllowOrigin, "*")
		writer.Header().Set(headers.AccessControlAllowCredentials, "true")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("upstream"))
	})
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From:           hosts.Parse("https://cors.local"),
			To:             backend.AsHost(),
			AllowedOrigins: config.AllowedOrigins{"http://localhost:*"},
			Mocks: config.Mocks{{
				Matcher:  config.RequestMatcher{Path: "/mock"},
				Response: config.Response{Code: http.StatusCreated, Raw: "mock"},
			}},
			Scripts: config.Scripts{{
				Matcher: config.RequestMatcher{Path: "/script"},
				Script:  `response:WriteHeader(202)`,
			}},
		}},
	})

	routes := []struct {
		name   string
		path   string
		status int
	}{
		{name: "proxy", path: "/api", status: http.StatusOK},
		{name: "mock", path: "/mock", status: http.StatusCreated},
		{name: "script", path: "/script", status: http.StatusAccepted},
	}

	for _, route := range routes {
		t.Run(route.name+" reflects an allowed origin", func(t *testing.T) {
			req := integration.NewRequest(t, http.MethodGet, env.URL("cors.local", route.path))
			req.Header.Set(headers.Origin, allowedOrigin)

			result := env.Do(t, req)

			assert.Equal(t, route.status, result.Response.StatusCode)
			assert.Equal(t, allowedOrigin, result.Response.Header.Get(headers.AccessControlAllowOrigin))
			assert.Contains(t, result.Response.Header.Get(headers.AccessControlExposeHeaders), "Date")
		})

		t.Run(route.name+" withholds CORS headers from another origin", func(t *testing.T) {
			req := integration.NewRequest(t, http.MethodGet, env.URL("cors.local", route.path))
			req.Header.Set(headers.Origin, rejectedOrigin)

			result := env.Do(t, req)

			assert.Equal(t, route.status, result.Response.StatusCode)
			assert.Empty(t, corsHeaderNames(result.Response.Header))
		})
	}

	t.Run("proxied request from another origin still reaches the backend", func(t *testing.T) {
		backend.Reset()

		req := integration.NewRequest(t, http.MethodGet, env.URL("cors.local", "/api"))
		req.Header.Set(headers.Origin, rejectedOrigin)

		result := env.Do(t, req)

		assert.True(t, result.HasBackendRequest())
		assert.Equal(t, "upstream", result.BodyString())
	})

	t.Run("preflight from another origin is forbidden", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodOptions, env.URL("cors.local", "/api"))
		req.Header.Set(headers.Origin, rejectedOrigin)
		req.Header.Set(headers.AccessControlRequestMethod, http.MethodPut)

		result := env.Do(t, req)

		assert.Equal(t, http.StatusForbidden, result.Response.StatusCode)
		assert.Empty(t, corsHeaderNames(result.Response.Header))
		assert.False(t, result.HasBackendRequest())
	})

	t.Run("preflight from an allowed origin is answered", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodOptions, env.URL("cors.local", "/api"))
		req.Header.Set(headers.Origin, allowedOrigin)

		result := env.Do(t, req)

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, allowedOrigin, result.Response.Header.Get(headers.AccessControlAllowOrigin))
	})
}
