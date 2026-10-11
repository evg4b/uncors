package script_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/handler/script"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testconstants"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const applicationJSON = "application/json"

type scriptTestCase struct {
	name           string
	script         string
	file           string
	request        func(*http.Request) *http.Request
	expectedStatus int
	expectedBody   string
	expectedHeader map[string]string
}

func serveScript(
	t *testing.T,
	scriptConfig *config.Script,
	fs afero.Fs,
	request *http.Request,
) (*httptest.ResponseRecorder, error) {
	t.Helper()

	handler := script.NewHandler(
		script.WithScript(scriptConfig),
		script.WithFileSystem(fs),
	)

	recorder := httptest.NewRecorder()

	return recorder, handler.ServeHTTP(server.NewResponseRecorder(recorder), request)
}

func runScriptTests(t *testing.T, fs afero.Fs, tests []scriptTestCase) {
	t.Helper()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test/path", nil)
			request.Header.Set(headers.UserAgent, "TestAgent/1.0")

			if testCase.request != nil {
				request = testCase.request(request)
			}

			scriptConfig := &config.Script{Script: testCase.script, File: testCase.file}

			recorder, err := serveScript(t, scriptConfig, fs, request)
			require.NoError(t, err)

			expectedStatus := testCase.expectedStatus
			if expectedStatus == 0 {
				expectedStatus = http.StatusOK
			}

			assert.Equal(t, expectedStatus, recorder.Code)
			assert.Equal(t, testCase.expectedBody, testutils.ReadBody(t, recorder))

			for key, value := range testCase.expectedHeader {
				assert.Equal(t, value, recorder.Header().Get(key))
			}
		})
	}
}

func withQuery(query string) func(*http.Request) *http.Request {
	return func(request *http.Request) *http.Request {
		request.URL.RawQuery = query

		return request
	}
}

func withHeader(key string, values ...string) func(*http.Request) *http.Request {
	return func(request *http.Request) *http.Request {
		for _, value := range values {
			request.Header.Add(key, value)
		}

		return request
	}
}

func TestScriptHandlerResponse(t *testing.T) {
	runScriptTests(t, afero.NewMemMapFs(), []scriptTestCase{
		{
			name:         "WriteString",
			script:       "response:WriteHeader(200)\nresponse:WriteString(\"Hello, \")\nresponse:WriteString(\"World!\")",
			expectedBody: "Hello, World!",
		},
		{
			name:           "Write with custom status code",
			script:         "response:WriteHeader(201)\nresponse:Write(\"Created resource\")",
			expectedStatus: http.StatusCreated,
			expectedBody:   "Created resource",
		},
		{
			name:         "multiple Write calls",
			script:       "response:WriteHeader(200)\nresponse:Write(\"Line 1\\n\")\nresponse:Write(\"Line 2\")",
			expectedBody: "Line 1\nLine 2",
		},
		{
			name:         "Write without explicit WriteHeader",
			script:       `response:Write("Auto status")`,
			expectedBody: "Auto status",
		},
		{
			name:         "WriteString without explicit WriteHeader",
			script:       `response:WriteString("Auto status")`,
			expectedBody: "Auto status",
		},
		{
			name:           "empty body",
			script:         "response:WriteHeader(204)",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:         "first WriteHeader wins",
			script:       "response:WriteHeader(200)\nresponse:WriteHeader(500)\nresponse:WriteString(\"First status wins\")",
			expectedBody: "First status wins",
		},
		{
			name:         "status field is not writable",
			script:       "response.status = 500\nresponse:WriteString(\"Status not writable\")",
			expectedBody: "Status not writable",
		},
		{
			name:         "body field is not writable",
			script:       "response.body = \"ignored\"\nresponse:WriteString(\"Actual body\")",
			expectedBody: "Actual body",
		},
		{
			name: "custom fields are allowed",
			script: `
response.custom_field = "custom_value"
response:WriteString("Custom: " .. response.custom_field)
`,
			expectedBody: "Custom: custom_value",
		},
	})
}

func TestScriptHandlerResponseHeaders(t *testing.T) {
	runScriptTests(t, afero.NewMemMapFs(), []scriptTestCase{
		{
			name: "set by index",
			script: `
response.headers["X-Custom-1"] = "Value1"
response.headers["X-Custom-2"] = "Value2"
response:WriteString(response.headers["X-Custom-1"])
`,
			expectedBody:   "Value1",
			expectedHeader: map[string]string{"X-Custom-1": "Value1", "X-Custom-2": "Value2"},
		},
		{
			name: "Header():Set and Header():Get",
			script: `
response:Header():Set("Content-Type", "application/json")
response:WriteString(response:Header():Get("Content-Type"))
`,
			expectedBody:   applicationJSON,
			expectedHeader: map[string]string{headers.ContentType: applicationJSON},
		},
		{
			name: "headers:Set and headers:Get",
			script: `
response.headers:Set("X-Custom", "CustomValue")
response:WriteString(response.headers:Get("X-Custom"))
`,
			expectedBody:   "CustomValue",
			expectedHeader: map[string]string{"X-Custom": "CustomValue"},
		},
		{
			name: "mixed index and method API",
			script: `
response:Header():Set("X-New-Style", "new")
response.headers["X-Old-Style"] = "old"
`,
			expectedHeader: map[string]string{"X-New-Style": "new", "X-Old-Style": "old"},
		},
	})
}

func TestScriptHandlerLibraries(t *testing.T) {
	runScriptTests(t, afero.NewMemMapFs(), []scriptTestCase{
		{
			name:         "math",
			script:       `response:WriteString(tostring(require("math").floor(3.7)))`,
			expectedBody: "3",
		},
		{
			name:         "string",
			script:       `response:WriteString(require("string").upper("hello"))`,
			expectedBody: "HELLO",
		},
		{
			name:         "table",
			script:       `response:WriteString(require("table").concat({"apple", "banana", "cherry"}, ", "))`,
			expectedBody: "apple, banana, cherry",
		},
		{
			name:         "json encode",
			script:       `response:WriteString(require("json").encode({message = "hello", count = 42}))`,
			expectedBody: `{"count":42,"message":"hello"}`,
		},
		{
			name: "json decode",
			script: `
local decoded = require("json").decode('{"name":"test","value":123}')
response:WriteString("Name: " .. decoded.name .. ", Value: " .. tostring(decoded.value))
`,
			expectedBody: "Name: test, Value: 123",
		},
	})
}

func TestScriptHandlerRequest(t *testing.T) {
	runScriptTests(t, afero.NewMemMapFs(), []scriptTestCase{
		{
			name:         "method",
			script:       `response:WriteString("Method: " .. request.method)`,
			expectedBody: "Method: GET",
		},
		{
			name:         "path",
			script:       `response:WriteString("Path: " .. request.path)`,
			expectedBody: "Path: /test/path",
		},
		{
			name:         "url",
			script:       `if request.url ~= nil then response:WriteString("URL exists") end`,
			expectedBody: "URL exists",
		},
		{
			name: "host",
			request: func(request *http.Request) *http.Request {
				request.Host = "example.com"

				return request
			},
			script:       `response:WriteString("Host: " .. request.host)`,
			expectedBody: "Host: example.com",
		},
		{
			name: "body",
			request: func(request *http.Request) *http.Request {
				return httptest.NewRequestWithContext(
					request.Context(), http.MethodPost, "/", strings.NewReader("test request body"),
				)
			},
			script:       `response:WriteString("Body: " .. request.body)`,
			expectedBody: "Body: test request body",
		},
		{
			name: "path parameters",
			request: func(request *http.Request) *http.Request {
				return testutils.SetMuxVars(request, map[string]string{"id": "123", "action": "edit"})
			},
			script: `
local params = request.path_params
response:WriteString("id: " .. params["id"] .. ", action: " .. params["action"])
`,
			expectedBody: "id: 123, action: edit",
		},
		{
			name:         "single value query parameter",
			request:      withQuery("name=John&other=1"),
			script:       `response:WriteString("Name: " .. request.query_params["name"])`,
			expectedBody: "Name: John",
		},
		{
			name:    "multi-value query parameter",
			request: withQuery("tags=go&tags=lua&tags=testing"),
			script: `
local tags = request.query_params["tags"]
response:WriteString("Tags: " .. table.concat(tags, ","))
`,
			expectedBody: "Tags: go,lua,testing",
		},
		{
			name:         "missing query parameter",
			script:       `if request.query_params["missing"] == nil then response:WriteString("No param") end`,
			expectedBody: "No param",
		},
		{
			name:         "single value header",
			script:       `response:WriteString("User-Agent: " .. request.headers["User-Agent"])`,
			expectedBody: "User-Agent: TestAgent/1.0",
		},
		{
			name:    "multi-value header",
			request: withHeader("Accept", "text/html", "application/json", "text/plain"),
			script: `
local accepts = request.headers["Accept"]
response:WriteString("Accept: " .. table.concat(accepts, ";"))
`,
			expectedBody: "Accept: text/html;application/json;text/plain",
		},
		{
			name:         "missing header",
			script:       `if request.headers["X-Missing"] == nil then response:WriteString("No header") end`,
			expectedBody: "No header",
		},
	})
}

func TestScriptHandlerFile(t *testing.T) {
	fs := testutils.FsFromMap(t, map[string]string{
		"test.lua":  "response:WriteHeader(200)\nresponse:WriteString(\"Hello from file\")",
		"error.lua": "response:WriteHeader(500)\nresponse:WriteString(\"Error response\")",
	})

	runScriptTests(t, fs, []scriptTestCase{
		{
			name:         "load script from file",
			file:         "test.lua",
			expectedBody: "Hello from file",
		},
		{
			name:           "load error script from file",
			file:           "error.lua",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Error response",
		},
	})
}

func TestScriptHandlerCORS(t *testing.T) {
	origin := hosts.Example.HTTP().String()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	request.Header.Set("Origin", origin)

	recorder, err := serveScript(t, &config.Script{Script: `response:WriteString("OK")`}, afero.NewMemMapFs(), request)
	require.NoError(t, err)

	assert.Equal(t, origin, recorder.Header().Get(headers.AccessControlAllowOrigin))
	assert.Equal(t, "true", recorder.Header().Get(headers.AccessControlAllowCredentials))
	assert.Equal(t, "*", recorder.Header().Get(headers.AccessControlAllowHeaders))
	assert.Equal(t, testconstants.AllMethods, recorder.Header().Get(headers.AccessControlAllowMethods))
}

func TestScriptHandlerErrors(t *testing.T) {
	tests := []struct {
		name   string
		script config.Script
	}{
		{name: "script file not found", script: config.Script{File: "nonexistent.lua"}},
		{name: "lua syntax error", script: config.Script{Script: "this is not valid lua code ###"}},
		{name: "lua runtime error", script: config.Script{Script: "local x = nil\nresponse:WriteString(x.field)"}},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

			_, err := serveScript(t, &testCase.script, afero.NewMemMapFs(), request)

			assert.Error(t, err)
		})
	}
}
