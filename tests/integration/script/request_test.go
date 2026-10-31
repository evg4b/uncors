//go:build integration

package script_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const echoScript = `
local json = require("json")
response:Header():Set("Content-Type", "application/json")
response:Header():Set("X-Script", "echo")
response:WriteHeader(200)
response:WriteString(json.encode({
  method = request.method,
  path = request.path,
  query = request.query,
  host = request.host,
  body = request.body,
  header = request.headers["X-Custom"],
  name = request.query_params["name"],
}))
`

func TestScriptRequestAndResponse(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://script.local"),
			To:   backend.AsHost(),
			Scripts: config.Scripts{
				{Matcher: config.RequestMatcher{Path: "/echo"}, Script: echoScript},
				{Matcher: config.RequestMatcher{Path: "/from-file"}, File: "/scripts/handler.lua"},
				{
					Matcher: config.RequestMatcher{Path: "/only-post", Method: http.MethodPost},
					Script:  `response:WriteHeader(201)`,
				},
			},
		}},
	}, integration.WithFile("/scripts/handler.lua", `
response:WriteHeader(200)
response:WriteString("from file")
`))

	t.Run("the script sees what the client sent and controls the response", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodPut, env.RawURL("script.local", "/echo?name=uncors&x=1"))
		req.Body = io.NopCloser(strings.NewReader(`{"a":1}`))
		req.ContentLength = 7
		req.Header.Set("X-Custom", "custom value")

		result := env.Do(t, req)

		require.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "application/json", result.Response.Header.Get("Content-Type"))
		assert.Equal(t, "echo", result.Response.Header.Get("X-Script"))
		assert.Equal(t, "*", result.Response.Header.Get("Access-Control-Allow-Origin"))
		assert.False(t, result.HasBackendRequest())

		var seen map[string]string
		require.NoError(t, json.Unmarshal([]byte(result.BodyString()), &seen))
		assert.Equal(t, map[string]string{
			"method": http.MethodPut,
			"path":   "/echo",
			"query":  "name=uncors&x=1",
			"host":   req.URL.Host,
			"body":   `{"a":1}`,
			"header": "custom value",
			"name":   "uncors",
		}, seen)
	})

	t.Run("a file script is run", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("script.local", "/from-file")))

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "from file", result.BodyString())
		assert.False(t, result.HasBackendRequest())
	})

	t.Run("a file script is read on each request", func(t *testing.T) {
		require.NoError(t, env.FS.Remove("/scripts/handler.lua"))
		require.NoError(t, integration.WithFile("/scripts/handler.lua", `response:WriteString("edited")`)(env.FS))

		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("script.local", "/from-file")))

		assert.Equal(t, "edited", result.BodyString())
	})

	t.Run("a method mismatch falls through to the backend", func(t *testing.T) {
		posted := env.Do(t, integration.NewRequest(t, http.MethodPost, env.URL("script.local", "/only-post")))
		fetched := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("script.local", "/only-post")))

		assert.Equal(t, http.StatusCreated, posted.Response.StatusCode)
		assert.False(t, posted.HasBackendRequest())
		assert.Equal(t, "ok", fetched.BodyString())
		assert.Contains(t, fetched.BackendRequest(t), "GET /only-post HTTP/1.1")
	})
}

func TestScriptFailures(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://script.local"),
			To:   backend.AsHost(),
			Scripts: config.Scripts{
				{Matcher: config.RequestMatcher{Path: "/error"}, Script: `error("boom")`},
				{Matcher: config.RequestMatcher{Path: "/syntax"}, Script: `response:WriteHeader(`},
				{Matcher: config.RequestMatcher{Path: "/exit"}, Script: `os.exit(1)`},
				{
					Matcher: config.RequestMatcher{Path: "/loop"},
					Script:  `while true do end`,
					Timeout: 200 * time.Millisecond,
				},
				{
					Matcher:     config.RequestMatcher{Path: "/upload"},
					Script:      `response:WriteString(request.body)`,
					MaxBodySize: 3,
				},
				{Matcher: config.RequestMatcher{Path: "/missing"}, File: "/scripts/missing.lua"},
				{Matcher: config.RequestMatcher{Path: "/silent"}, Script: `local unused = 1`},
				{
					Matcher: config.RequestMatcher{Path: "/late-error"},
					Script:  "response:WriteHeader(200)\nerror(\"late\")",
				},
			},
		}},
	}, integration.WithFile("/scripts/missing.lua", `response:WriteHeader(200)`))

	require.NoError(t, env.FS.Remove("/scripts/missing.lua"))

	tests := []struct {
		name    string
		path    string
		body    string
		status  int
		message string
	}{
		{
			name:    "runtime error",
			path:    "/error",
			status:  http.StatusInternalServerError,
			message: "script error: <string>:1: boom",
		},
		{name: "syntax error", path: "/syntax", status: http.StatusInternalServerError, message: "syntax error"},
		{
			name:    "os.exit",
			path:    "/exit",
			status:  http.StatusInternalServerError,
			message: "os.exit is not supported in scripts",
		},
		{name: "timeout", path: "/loop", status: http.StatusInternalServerError, message: "script timed out"},
		{
			name:    "body over the limit",
			path:    "/upload",
			body:    "hello",
			status:  http.StatusInternalServerError,
			message: "request body is too large for a script: limit is 3 bytes",
		},
		{
			name:    "file removed after start",
			path:    "/missing",
			status:  http.StatusInternalServerError,
			message: "script file not found",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name+" is reported to the client", func(t *testing.T) {
			req := integration.NewRequest(t, http.MethodPost, env.URL("script.local", testCase.path))
			req.Body = io.NopCloser(strings.NewReader(testCase.body))
			req.ContentLength = int64(len(testCase.body))
			req.Header.Set("Origin", "https://front.local")

			result := env.Do(t, req)

			assert.Equal(t, testCase.status, result.Response.StatusCode)
			assert.Contains(t, result.BodyString(), "An error occurred: ")
			assert.Contains(t, result.BodyString(), testCase.message)
			assert.Equal(t, "https://front.local", result.Response.Header.Get("Access-Control-Allow-Origin"))
			assert.False(t, result.HasBackendRequest())
		})
	}

	t.Run("a script that writes nothing answers 200 with an empty body", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("script.local", "/silent")))

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Empty(t, result.BodyString())
	})

	t.Run("an error after the status line was sent aborts the response", func(t *testing.T) {
		resp, err := env.Client.Do(integration.NewRequest(t, http.MethodGet, env.URL("script.local", "/late-error")))
		if err == nil {
			_, err = io.ReadAll(resp.Body)
			resp.Body.Close()
		}

		require.Error(t, err)
	})
}
