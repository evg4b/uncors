package mock_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/mock"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	logger := mocks.NewNoopLogger(t)

	t.Run("content type setting", func(t *testing.T) {
		tests := []struct {
			name     string
			body     string
			expected string
		}{
			{
				name:     "plain text",
				body:     `status: ok`,
				expected: "text/plain; charset=utf-8",
			},
			{
				name:     "json",
				body:     `{ "status": "ok" }`,
				expected: "text/plain; charset=utf-8",
			},
			{
				name:     "html",
				body:     `<html></html>`,
				expected: "text/html; charset=utf-8",
			},
			{
				name:     "xml",
				body:     `<?xml />`,
				expected: "text/xml; charset=utf-8",
			},
			{
				name:     "png",
				body:     "\x89PNG\x0D\x0A\x1A\x0A",
				expected: "image/png",
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := mock.NewMockHandler(mock.WithLogger(logger), mock.WithResponse(mock.Response{
					Code:       200,
					RawContent: testCase.body,
				}))

				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "/", nil)
				handler.ServeHTTP(recorder, request)

				header := testutils.ReadHeader(t, recorder)
				assert.EqualValues(t, testCase.expected, header.Get("Content-Type"))
			})
		}
	})

	t.Run("headers settings", func(t *testing.T) {
		tests := []struct {
			name     string
			response mock.Response
			expected http.Header
		}{
			{
				name: "should put default CORS headers",
				response: mock.Response{
					Code:       200,
					RawContent: "test content",
				},
				expected: map[string][]string{
					"Access-Control-Allow-Origin":      {"*"},
					"Access-Control-Allow-Credentials": {"true"},
					"Content-Type":                     {"text/plain; charset=utf-8"},
					"Access-Control-Allow-Methods": {
						"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
					},
				},
			},
			{
				name: "should set response code",
				response: mock.Response{
					Code:       200,
					RawContent: "test content",
				},
				expected: map[string][]string{
					"Access-Control-Allow-Origin":      {"*"},
					"Access-Control-Allow-Credentials": {"true"},
					"Content-Type":                     {"text/plain; charset=utf-8"},
					"Access-Control-Allow-Methods": {
						"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
					},
				},
			},
			{
				name: "should set custom headers",
				response: mock.Response{
					Code: 200,
					Headers: map[string]string{
						"X-Key": "X-Key-Value",
					},
					RawContent: "test content",
				},
				expected: map[string][]string{
					"Access-Control-Allow-Origin":      {"*"},
					"Access-Control-Allow-Credentials": {"true"},
					"Content-Type":                     {"text/plain; charset=utf-8"},
					"X-Key":                            {"X-Key-Value"},
					"Access-Control-Allow-Methods": {
						"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
					},
				},
			},
			{
				name: "should override default headers",
				response: mock.Response{
					Code: 200,
					Headers: map[string]string{
						"Access-Control-Allow-Origin":      "localhost",
						"Access-Control-Allow-Credentials": "false",
						"Content-Type":                     "none",
					},
					RawContent: "test content",
				},
				expected: map[string][]string{
					"Access-Control-Allow-Origin":      {"localhost"},
					"Access-Control-Allow-Credentials": {"false"},
					"Content-Type":                     {"none"},
					"Access-Control-Allow-Methods": {
						"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
					},
				},
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := mock.NewMockHandler(
					mock.WithResponse(testCase.response),
					mock.WithLogger(logger),
				)

				request := httptest.NewRequest(http.MethodGet, "/", nil)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)

				assert.EqualValues(t, testCase.expected, testutils.ReadHeader(t, recorder))
				assert.Equal(t, 200, recorder.Code)
			})
		}
	})

	t.Run("status code", func(t *testing.T) {
		tests := []struct {
			name     string
			response mock.Response
			expected int
		}{
			{
				name: "provide 201 code",
				response: mock.Response{
					Code: 201,
				},
				expected: 201,
			},
			{
				name: "provide 503 code",
				response: mock.Response{
					Code: 503,
				},
				expected: 503,
			},
			{
				name:     "automatically provide 200 code",
				response: mock.Response{},
				expected: 200,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := mock.NewMockHandler(
					mock.WithResponse(testCase.response),
					mock.WithLogger(logger),
				)

				request := httptest.NewRequest(http.MethodGet, "/", nil)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)

				assert.Equal(t, testCase.expected, recorder.Code)
			})
		}
	})
}
