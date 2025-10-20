package script_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/charmbracelet/log"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/script"
	"github.com/evg4b/uncors/testing/testconstants"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScriptHandler(t *testing.T) {
	t.Run("inline script execution", func(t *testing.T) {
		tests := []struct {
			name           string
			script         string
			expectedStatus int
			expectedBody   string
			expectedHeader map[string]string
		}{
			{
				name: "simple response",
				script: `
response.status = 200
response.body = "Hello from Lua"
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "Hello from Lua",
			},
			{
				name: "json response",
				script: `
response.status = 200
response.body = '{"message": "success"}'
response.headers["Content-Type"] = "application/json"
`,
				expectedStatus: http.StatusOK,
				expectedBody:   `{"message": "success"}`,
				expectedHeader: map[string]string{
					headers.ContentType: "application/json",
				},
			},
			{
				name: "custom status code",
				script: `
response.status = 201
response.body = "Created"
`,
				expectedStatus: http.StatusCreated,
				expectedBody:   "Created",
			},
			{
				name: "access request method",
				script: `
response.status = 200
response.body = "Method: " .. request.method
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "Method: GET",
			},
			{
				name: "access request path",
				script: `
response.status = 200
response.body = "Path: " .. request.path
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "Path: /test/path",
			},
			{
				name: "access request headers",
				script: `
response.status = 200
response.body = "User-Agent: " .. request.headers["User-Agent"]
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "User-Agent: TestAgent/1.0",
			},
			{
				name: "use math library",
				script: `
local math = require("math")
response.status = 200
response.body = tostring(math.floor(3.7))
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "3",
			},
			{
				name: "use string library",
				script: `
local string = require("string")
response.status = 200
response.body = string.upper("hello")
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "HELLO",
			},
			{
				name: "multiple custom headers",
				script: `
response.status = 200
response.body = "Test"
response.headers["X-Custom-1"] = "Value1"
response.headers["X-Custom-2"] = "Value2"
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "Test",
				expectedHeader: map[string]string{
					"X-Custom-1": "Value1",
					"X-Custom-2": "Value2",
				},
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := script.NewHandler(
					script.WithLogger(log.New(io.Discard)),
					script.WithScript(config.Script{
						Script: testCase.script,
					}),
					script.WithFileSystem(testutils.FsFromMap(t, map[string]string{})),
				)

				req := httptest.NewRequest(http.MethodGet, "/test/path", nil)
				req.Header.Set("User-Agent", "TestAgent/1.0")
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

				assert.Equal(t, testCase.expectedStatus, recorder.Code)
				assert.Equal(t, testCase.expectedBody, testutils.ReadBody(t, recorder))

				if testCase.expectedHeader != nil {
					for key, value := range testCase.expectedHeader {
						assert.Equal(t, value, recorder.Header().Get(key))
					}
				}
			})
		}
	})

	t.Run("file-based script execution", func(t *testing.T) {
		fileSystem := testutils.FsFromMap(t, map[string]string{
			"test.lua": `
response.status = 200
response.body = "Hello from file"
`,
			"error.lua": `
response.status = 500
response.body = "Error response"
`,
		})

		tests := []struct {
			name           string
			file           string
			expectedStatus int
			expectedBody   string
		}{
			{
				name:           "load script from file",
				file:           "test.lua",
				expectedStatus: http.StatusOK,
				expectedBody:   "Hello from file",
			},
			{
				name:           "load error script from file",
				file:           "error.lua",
				expectedStatus: http.StatusInternalServerError,
				expectedBody:   "Error response",
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := script.NewHandler(
					script.WithLogger(log.New(io.Discard)),
					script.WithScript(config.Script{
						File: testCase.file,
					}),
					script.WithFileSystem(fileSystem),
				)

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

				assert.Equal(t, testCase.expectedStatus, recorder.Code)
				assert.Equal(t, testCase.expectedBody, testutils.ReadBody(t, recorder))
			})
		}
	})

	t.Run("request object properties", func(t *testing.T) {
		tests := []struct {
			name         string
			setupRequest func(*http.Request)
			script       string
			expectedBody string
		}{
			{
				name: "access query parameters",
				setupRequest: func(r *http.Request) {
					q := r.URL.Query()
					q.Add("param1", "value1")
					q.Add("param2", "value2")
					r.URL.RawQuery = q.Encode()
				},
				script: `
response.status = 200
response.body = "param1: " .. request.query_params["param1"]
`,
				expectedBody: "param1: value1",
			},
			{
				name: "access host",
				setupRequest: func(r *http.Request) {
					r.Host = "example.com"
				},
				script: `
response.status = 200
response.body = "Host: " .. request.host
`,
				expectedBody: "Host: example.com",
			},
			{
				name: "access URL",
				setupRequest: func(_ *http.Request) {
					// URL is set by NewRequest
				},
				script: `
response.status = 200
local url = request.url
if url ~= nil then
    response.body = "URL exists"
else
    response.body = "URL missing"
end
`,
				expectedBody: "URL exists",
			},
			{
				name: "access request body",
				setupRequest: func(_ *http.Request) {
					// Body will be set in the test
				},
				script: `
response.status = 200
response.body = "Body: " .. request.body
`,
				expectedBody: "Body: test request body",
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				var req *http.Request
				if testCase.name == "access request body" {
					req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("test request body"))
				} else {
					req = httptest.NewRequest(http.MethodGet, "/", nil)
				}
				testCase.setupRequest(req)

				handler := script.NewHandler(
					script.WithLogger(log.New(io.Discard)),
					script.WithScript(config.Script{
						Script: testCase.script,
					}),
					script.WithFileSystem(testutils.FsFromMap(t, map[string]string{})),
				)

				recorder := httptest.NewRecorder()
				handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

				assert.Equal(t, testCase.expectedBody, testutils.ReadBody(t, recorder))
			})
		}
	})

	t.Run("CORS headers", func(t *testing.T) {
		handler := script.NewHandler(
			script.WithLogger(log.New(io.Discard)),
			script.WithScript(config.Script{
				Script: `
response.status = 200
response.body = "OK"
`,
			}),
			script.WithFileSystem(testutils.FsFromMap(t, map[string]string{})),
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "http://example.com")
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

		// Check CORS headers are set (when Origin is set, it should be returned)
		assert.Equal(t, "http://example.com", recorder.Header().Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "true", recorder.Header().Get(headers.AccessControlAllowCredentials))
		assert.Equal(t, "*", recorder.Header().Get(headers.AccessControlAllowHeaders))
		assert.Equal(t, testconstants.AllMethods, recorder.Header().Get(headers.AccessControlAllowMethods))
	})

	t.Run("error handling", func(t *testing.T) {
		tests := []struct {
			name         string
			script       config.Script
			expectedCode int
		}{
			{
				name:   "script not defined",
				script: config.Script{
					// Empty script
				},
				expectedCode: http.StatusInternalServerError,
			},
			{
				name: "both script and file defined",
				script: config.Script{
					Script: "response.status = 200",
					File:   "test.lua",
				},
				expectedCode: http.StatusInternalServerError,
			},
			{
				name: "script file not found",
				script: config.Script{
					File: "nonexistent.lua",
				},
				expectedCode: http.StatusInternalServerError,
			},
			{
				name: "lua syntax error",
				script: config.Script{
					Script: "this is not valid lua code ###",
				},
				expectedCode: http.StatusInternalServerError,
			},
			{
				name: "lua runtime error",
				script: config.Script{
					Script: `
local x = nil
response.body = x.field  -- This will cause an error
`,
				},
				expectedCode: http.StatusInternalServerError,
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := script.NewHandler(
					script.WithLogger(log.New(io.Discard)),
					script.WithScript(testCase.script),
					script.WithFileSystem(testutils.FsFromMap(t, map[string]string{})),
				)

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

				assert.Equal(t, testCase.expectedCode, recorder.Code)
			})
		}
	})

	t.Run("default response status", func(t *testing.T) {
		handler := script.NewHandler(
			script.WithLogger(log.New(io.Discard)),
			script.WithScript(config.Script{
				Script: `
-- Don't set status, should default to 200
response.body = "Default status"
`,
			}),
			script.WithFileSystem(testutils.FsFromMap(t, map[string]string{})),
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "Default status", testutils.ReadBody(t, recorder))
	})

	t.Run("empty response body", func(t *testing.T) {
		handler := script.NewHandler(
			script.WithLogger(log.New(io.Discard)),
			script.WithScript(config.Script{
				Script: `
response.status = 204
-- Don't set body
`,
			}),
			script.WithFileSystem(testutils.FsFromMap(t, map[string]string{})),
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

		assert.Equal(t, http.StatusNoContent, recorder.Code)
		assert.Empty(t, testutils.ReadBody(t, recorder))
	})

	t.Run("complex script with table library", func(t *testing.T) {
		handler := script.NewHandler(
			script.WithLogger(log.New(io.Discard)),
			script.WithScript(config.Script{
				Script: `
local table = require("table")
local items = {"apple", "banana", "cherry"}
local result = table.concat(items, ", ")
response.status = 200
response.body = result
`,
			}),
			script.WithFileSystem(testutils.FsFromMap(t, map[string]string{})),
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "apple, banana, cherry", testutils.ReadBody(t, recorder))
	})
}

func TestScriptHandlerOptions(t *testing.T) {
	t.Run("WithLogger", func(t *testing.T) {
		logger := log.New(io.Discard)
		handler := script.NewHandler(script.WithLogger(logger))
		require.NotNil(t, handler)
	})

	t.Run("WithScript", func(t *testing.T) {
		scriptConfig := config.Script{Script: "response.status = 200"}
		handler := script.NewHandler(script.WithScript(scriptConfig))
		require.NotNil(t, handler)
	})

	t.Run("WithFileSystem", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{})
		handler := script.NewHandler(script.WithFileSystem(fs))
		require.NotNil(t, handler)
	})

	t.Run("all options together", func(t *testing.T) {
		logger := log.New(io.Discard)
		scriptConfig := config.Script{Script: "response.status = 200"}
		fs := testutils.FsFromMap(t, map[string]string{})

		handler := script.NewHandler(
			script.WithLogger(logger),
			script.WithScript(scriptConfig),
			script.WithFileSystem(fs),
		)

		require.NotNil(t, handler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}
