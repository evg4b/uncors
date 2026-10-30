//go:build integration

package cors_test

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

const requestedHeaders = "content-type, authorization"

func preflight(t *testing.T, env *integration.Env, host, path, origin string) *integration.Result {
	t.Helper()

	req := integration.NewRequest(t, http.MethodOptions, env.URL(host, path))
	req.Header.Set(headers.Origin, origin)
	req.Header.Set(headers.AccessControlRequestMethod, http.MethodPost)
	req.Header.Set(headers.AccessControlRequestHeaders, requestedHeaders)

	return env.Do(t, req)
}

func TestPreflightOnMocksAndScripts(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{
			{
				From:           hosts.Parse("http://preflight.local"),
				To:             backend.AsHost(),
				AllowedOrigins: config.AllowedOrigins{"http://localhost:*"},
				Mocks: config.Mocks{
					{
						Matcher:  config.RequestMatcher{Path: "/mock"},
						Response: config.Response{Code: http.StatusCreated, Raw: "mock"},
					},
					{
						Matcher:  config.RequestMatcher{Path: "/options-mock", Method: http.MethodOptions},
						Response: config.Response{Code: http.StatusAccepted, Raw: "options mock"},
					},
				},
				Scripts: config.Scripts{{
					Matcher: config.RequestMatcher{Path: "/script"},
					Script:  `response:WriteHeader(202)`,
				}},
			},
			{
				From:            hosts.Parse("http://disabled.local"),
				To:              backend.AsHost(),
				OptionsHandling: config.OptionsHandling{Disabled: true},
				Mocks: config.Mocks{{
					Matcher:  config.RequestMatcher{Path: "/mock"},
					Response: config.Response{Code: http.StatusCreated, Raw: "mock"},
				}},
			},
		},
	})

	for _, path := range []string{"/mock", "/script"} {
		t.Run(path+" answers a preflight with the requested headers", func(t *testing.T) {
			result := preflight(t, env, "preflight.local", path, allowedOrigin)

			assert.Equal(t, http.StatusOK, result.Response.StatusCode)
			assert.Equal(t, allowedOrigin, result.Response.Header.Get(headers.AccessControlAllowOrigin))
			assert.Equal(t, http.MethodPost, result.Response.Header.Get(headers.AccessControlAllowMethods))
			assert.Equal(t, requestedHeaders, result.Response.Header.Get(headers.AccessControlAllowHeaders))
			assert.Empty(t, result.BodyString())
			assert.False(t, result.HasBackendRequest())
		})

		t.Run(path+" forbids a preflight from another origin", func(t *testing.T) {
			result := preflight(t, env, "preflight.local", path, rejectedOrigin)

			assert.Equal(t, http.StatusForbidden, result.Response.StatusCode)
			assert.Empty(t, corsHeaderNames(result.Response.Header))
		})
	}

	t.Run("a mock for the OPTIONS method answers the preflight itself", func(t *testing.T) {
		result := preflight(t, env, "preflight.local", "/options-mock", allowedOrigin)

		assert.Equal(t, http.StatusAccepted, result.Response.StatusCode)
		assert.Equal(t, "options mock", result.BodyString())
	})

	t.Run("a mock answers the preflight when options handling is disabled", func(t *testing.T) {
		result := preflight(t, env, "disabled.local", "/mock", allowedOrigin)

		assert.Equal(t, http.StatusCreated, result.Response.StatusCode)
		assert.Equal(t, "mock", result.BodyString())
	})
}
