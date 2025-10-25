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

const applicationJSON = "application/json"

type scriptTestCase struct {
	name           string
	script         string
	expectedStatus int
	expectedBody   string
	expectedHeader map[string]string
}

func runScriptTests(t *testing.T, tests []scriptTestCase) {
	t.Helper()
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
			req.Header.Set(headers.UserAgent, "TestAgent/1.0")
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
}

func TestScriptHandler(t *testing.T) { // nolint:cyclop, gocognit
	t.Run("inline script execution", func(t *testing.T) {
		tests := []scriptTestCase{
			{
				name: "simple response",
				script: `
response:WriteHeader(200)
response:WriteString("Hello from Lua")
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "Hello from Lua",
			},
			{
				name: "json response",
				script: `
response.headers["Content-Type"] = "` + applicationJSON + `"
response:WriteHeader(200)
response:WriteString('{"message": "success"}')
`,
				expectedStatus: http.StatusOK,
				expectedBody:   `{"message": "success"}`,
				expectedHeader: map[string]string{
					headers.ContentType: applicationJSON,
				},
			},
			{
				name: "custom status code",
				script: `
response:WriteHeader(201)
response:WriteString("Created")
`,
				expectedStatus: http.StatusCreated,
				expectedBody:   "Created",
			},
			{
				name: "access request method",
				script: `
response:WriteHeader(200)
response:WriteString("Method: " .. request.method)
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "Method: GET",
			},
			{
				name: "access request path",
				script: `
response:WriteHeader(200)
response:WriteString("Path: " .. request.path)
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "Path: /test/path",
			},
			{
				name: "access request headers",
				script: `
response:WriteHeader(200)
response:WriteString("User-Agent: " .. request.headers["User-Agent"])
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "User-Agent: TestAgent/1.0",
			},
			{
				name: "use math library",
				script: `
local math = require("math")
response:WriteHeader(200)
response:WriteString(tostring(math.floor(3.7)))
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "3",
			},
			{
				name: "use string library",
				script: `
local string = require("string")
response:WriteHeader(200)
response:WriteString(string.upper("hello"))
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "HELLO",
			},
			{
				name: "use json library",
				script: `
local json = require("json")
local data = {message = "hello", count = 42}
local encoded = json.encode(data)
response.headers["Content-Type"] = "` + applicationJSON + `"
response:WriteHeader(200)
response:WriteString(encoded)
`,
				expectedStatus: http.StatusOK,
				expectedBody:   `{"count":42,"message":"hello"}`,
				expectedHeader: map[string]string{
					headers.ContentType: applicationJSON,
				},
			},
			{
				name: "use json decode",
				script: `
local json = require("json")
local decoded = json.decode('{"name":"test","value":123}')
response:WriteHeader(200)
response:WriteString("Name: " .. decoded.name .. ", Value: " .. tostring(decoded.value))
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "Name: test, Value: 123",
			},
			{
				name: "multiple custom headers",
				script: `
response.headers["X-Custom-1"] = "Value1"
response.headers["X-Custom-2"] = "Value2"
response:WriteHeader(200)
response:WriteString("Test")
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "Test",
				expectedHeader: map[string]string{
					"X-Custom-1": "Value1",
					"X-Custom-2": "Value2",
				},
			},
		}

		runScriptTests(t, tests)
	})

	t.Run("file-based script execution", func(t *testing.T) {
		fileSystem := testutils.FsFromMap(t, map[string]string{
			"test.lua": `
response:WriteHeader(200)
response:WriteString("Hello from file")
`,
			"error.lua": `
response:WriteHeader(500)
response:WriteString("Error response")
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
response:WriteHeader(200)
response:WriteString("param1: " .. request.query_params["param1"])
`,
				expectedBody: "param1: value1",
			},
			{
				name: "access host",
				setupRequest: func(r *http.Request) {
					r.Host = "example.com"
				},
				script: `
response:WriteHeader(200)
response:WriteString("Host: " .. request.host)
`,
				expectedBody: "Host: example.com",
			},
			{
				name: "access URL",
				setupRequest: func(_ *http.Request) {
					// URL is set by NewRequest
				},
				script: `
response:WriteHeader(200)
local url = request.url
if url ~= nil then
    response:WriteString("URL exists")
else
    response:WriteString("URL missing")
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
response:WriteHeader(200)
response:WriteString("Body: " .. request.body)
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

	t.Run("path parameters", func(t *testing.T) {
		handler := script.NewHandler(
			script.WithLogger(log.New(io.Discard)),
			script.WithScript(config.Script{
				Script: `
response:WriteHeader(200)
local id = request.path_params["id"] or "none"
local action = request.path_params["action"] or "none"
response:WriteString("id: " .. id .. ", action: " .. action)
`,
			}),
			script.WithFileSystem(testutils.FsFromMap(t, map[string]string{})),
		)

		req := httptest.NewRequest(http.MethodGet, "/users/123/edit", nil)
		req = testutils.SetMuxVars(req, map[string]string{
			"id":     "123",
			"action": "edit",
		})
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "id: 123, action: edit", testutils.ReadBody(t, recorder))
	})

	t.Run("CORS headers", func(t *testing.T) {
		handler := script.NewHandler(
			script.WithLogger(log.New(io.Discard)),
			script.WithScript(config.Script{
				Script: `
response:WriteHeader(200)
response:WriteString("OK")
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
response:WriteString(x.field)  -- This will cause an error
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
response:WriteString("Default status")
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
response:WriteHeader(204)
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
response:WriteHeader(200)
response:WriteString(result)
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

	t.Run("Go-style API methods", func(t *testing.T) {
		tests := []struct {
			name           string
			script         string
			expectedStatus int
			expectedBody   string
			expectedHeader map[string]string
		}{
			{
				name: "WriteString method",
				script: `
response:WriteHeader(200)
response:WriteString("Hello, ")
response:WriteString("World!")
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "Hello, World!",
			},
			{
				name: "Write method",
				script: `
response:WriteHeader(201)
response:Write("Created resource")
`,
				expectedStatus: http.StatusCreated,
				expectedBody:   "Created resource",
			},
			{
				name: "Header().Set() method",
				script: `
response:Header():Set("Content-Type", "application/json")
response:Header():Set("X-Custom-Header", "CustomValue")
response:WriteHeader(200)
response:WriteString('{"status":"ok"}')
`,
				expectedStatus: http.StatusOK,
				expectedBody:   `{"status":"ok"}`,
				expectedHeader: map[string]string{
					headers.ContentType: applicationJSON,
					"X-Custom-Header":   "CustomValue",
				},
			},
			{
				name: "multiple Write calls",
				script: `
response:WriteHeader(200)
response:Write("Line 1\n")
response:Write("Line 2\n")
response:Write("Line 3")
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "Line 1\nLine 2\nLine 3",
			},
			{
				name: "Header().Get() method",
				script: `
response:Header():Set("X-Test", "TestValue")
local value = response:Header():Get("X-Test")
response:WriteHeader(200)
response:WriteString("Header value: " .. value)
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "Header value: TestValue",
			},
			{
				name: "mixed old and new API",
				script: `
response:Header():Set("X-New-Style", "new")
response.headers["X-Old-Style"] = "old"
response:WriteHeader(200)
response:WriteString("Mixed: ")
response:WriteString("old and new")
`,
				expectedStatus: http.StatusOK,
				expectedBody:   "Mixed: old and new",
				expectedHeader: map[string]string{
					"X-New-Style": "new",
					"X-Old-Style": "old",
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

				req := httptest.NewRequest(http.MethodGet, "/", nil)
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
}

func TestScriptHandlerOptions(t *testing.T) {
	t.Run("WithLogger", func(t *testing.T) {
		logger := log.New(io.Discard)
		handler := script.NewHandler(script.WithLogger(logger))
		require.NotNil(t, handler)
	})

	t.Run("WithScript", func(t *testing.T) {
		scriptConfig := config.Script{Script: "response:WriteHeader(200)"}
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
		scriptConfig := config.Script{Script: "response:WriteHeader(200)"}
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

func TestScriptHandler_MultiValueHeadersAndQueryParams(t *testing.T) {
	t.Run("multi-value query parameters", func(t *testing.T) {
		tests := []struct {
			name         string
			queryParams  map[string][]string
			script       string
			expectedBody string
		}{
			{
				name: "single value query param",
				queryParams: map[string][]string{
					"name": {"John"},
				},
				script: `
response:WriteHeader(200)
response:WriteString("Name: " .. request.query_params["name"])
`,
				expectedBody: "Name: John",
			},
			{
				name: "multi-value query param as table",
				queryParams: map[string][]string{
					"tags": {"go", "lua", "testing"},
				},
				script: `
response:WriteHeader(200)
local tags = request.query_params["tags"]
local result = ""
for i = 1, #tags do
    if i > 1 then result = result .. "," end
    result = result .. tags[i]
end
response:WriteString("Tags: " .. result)
`,
				expectedBody: "Tags: go,lua,testing",
			},
			{
				name:        "empty query params",
				queryParams: map[string][]string{},
				script: `
response:WriteHeader(200)
local name = request.query_params["missing"]
if name == nil then
    response:WriteString("No param")
else
    response:WriteString("Has param")
end
`,
				expectedBody: "No param",
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := script.NewHandler(
					script.WithLogger(log.New(io.Discard)),
					script.WithScript(config.Script{Script: testCase.script}),
				)

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				parsedQuery := req.URL.Query()
				for key, values := range testCase.queryParams {
					for _, value := range values {
						parsedQuery.Add(key, value)
					}
				}
				req.URL.RawQuery = parsedQuery.Encode()

				recorder := httptest.NewRecorder()
				handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, testCase.expectedBody, testutils.ReadBody(t, recorder))
			})
		}
	})

	t.Run("multi-value request headers", func(t *testing.T) {
		tests := []struct {
			name         string
			headers      map[string][]string
			script       string
			expectedBody string
		}{
			{
				name: "single value header",
				headers: map[string][]string{
					"X-Custom": {"Value1"},
				},
				script: `
response:WriteHeader(200)
response:WriteString("Header: " .. request.headers["X-Custom"])
`,
				expectedBody: "Header: Value1",
			},
			{
				name: "multi-value header as table",
				headers: map[string][]string{
					"Accept": {"text/html", "application/json", "text/plain"},
				},
				script: `
response:WriteHeader(200)
local accepts = request.headers["Accept"]
local result = ""
for i = 1, #accepts do
    if i > 1 then result = result .. ";" end
    result = result .. accepts[i]
end
response:WriteString("Accept: " .. result)
`,
				expectedBody: "Accept: text/html;application/json;text/plain",
			},
			{
				name:    "missing header",
				headers: map[string][]string{},
				script: `
response:WriteHeader(200)
local header = request.headers["X-Missing"]
if header == nil then
    response:WriteString("No header")
else
    response:WriteString("Has header")
end
`,
				expectedBody: "No header",
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := script.NewHandler(
					script.WithLogger(log.New(io.Discard)),
					script.WithScript(config.Script{Script: testCase.script}),
				)

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				for key, values := range testCase.headers {
					for _, value := range values {
						req.Header.Add(key, value)
					}
				}

				recorder := httptest.NewRecorder()
				handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, testCase.expectedBody, testutils.ReadBody(t, recorder))
			})
		}
	})
}

func TestScriptHandler_ResponseMetatableEdgeCases(t *testing.T) {
	t.Run("response metatable prevents status field writes", func(t *testing.T) {
		handler := script.NewHandler(
			script.WithLogger(log.New(io.Discard)),
			script.WithScript(config.Script{
				Script: `
response.status = 500
response:WriteHeader(200)
response:WriteString("Status not writable")
`,
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "Status not writable", testutils.ReadBody(t, recorder))
	})

	t.Run("response metatable prevents body field writes", func(t *testing.T) {
		handler := script.NewHandler(
			script.WithLogger(log.New(io.Discard)),
			script.WithScript(config.Script{
				Script: `
response.body = "This should be ignored"
response:WriteHeader(200)
response:WriteString("Actual body")
`,
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "Actual body", testutils.ReadBody(t, recorder))
	})

	t.Run("response metatable allows custom fields", func(t *testing.T) {
		handler := script.NewHandler(
			script.WithLogger(log.New(io.Discard)),
			script.WithScript(config.Script{
				Script: `
response.custom_field = "custom_value"
response:WriteHeader(200)
response:WriteString("Custom: " .. response.custom_field)
`,
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "Custom: custom_value", testutils.ReadBody(t, recorder))
	})

	t.Run("response WriteHeader idempotency", func(t *testing.T) {
		handler := script.NewHandler(
			script.WithLogger(log.New(io.Discard)),
			script.WithScript(config.Script{
				Script: `
response:WriteHeader(200)
response:WriteHeader(500)
response:WriteString("First status wins")
`,
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "First status wins", testutils.ReadBody(t, recorder))
	})

	t.Run("response Write without explicit WriteHeader", func(t *testing.T) {
		handler := script.NewHandler(
			script.WithLogger(log.New(io.Discard)),
			script.WithScript(config.Script{
				Script: `
response:Write("Auto status")
`,
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "Auto status", testutils.ReadBody(t, recorder))
	})

	t.Run("response WriteString without explicit WriteHeader", func(t *testing.T) {
		handler := script.NewHandler(
			script.WithLogger(log.New(io.Discard)),
			script.WithScript(config.Script{
				Script: `
response:WriteString("Auto status with string")
`,
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "Auto status with string", testutils.ReadBody(t, recorder))
	})

	t.Run("response headers metatable Get method", func(t *testing.T) {
		handler := script.NewHandler(
			script.WithLogger(log.New(io.Discard)),
			script.WithScript(config.Script{
				Script: `
response.headers["X-Test"] = "TestValue"
local value = response.headers:Get("X-Test")
response:WriteHeader(200)
response:WriteString("Value: " .. value)
`,
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "Value: TestValue", testutils.ReadBody(t, recorder))
	})

	t.Run("response headers metatable Set method", func(t *testing.T) {
		handler := script.NewHandler(
			script.WithLogger(log.New(io.Discard)),
			script.WithScript(config.Script{
				Script: `
response.headers:Set("X-Custom", "CustomValue")
response:WriteHeader(200)
response:WriteString("Header set")
`,
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "CustomValue", recorder.Header().Get("X-Custom"))
	})

	t.Run("response headers access via index", func(t *testing.T) {
		handler := script.NewHandler(
			script.WithLogger(log.New(io.Discard)),
			script.WithScript(config.Script{
				Script: `
response.headers["Content-Type"] = "text/plain"
local ct = response.headers["Content-Type"]
response:WriteHeader(200)
response:WriteString("CT: " .. ct)
`,
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "CT: text/plain", testutils.ReadBody(t, recorder))
	})
}
