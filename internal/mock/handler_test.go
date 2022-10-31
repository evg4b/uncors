package mock_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/mock"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

const textPlain = "text/plain; charset=utf-8"

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
				expected: textPlain,
			},
			{
				name:     "json",
				body:     `{ "status": "ok" }`,
				expected: textPlain,
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
				assert.EqualValues(t, testCase.expected, header.Get(headers.ContentType))
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
					headers.AccessControlAllowOrigin:      {"*"},
					headers.AccessControlAllowCredentials: {"true"},
					headers.ContentType:                   {textPlain},
					headers.AccessControlAllowMethods: {
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
					headers.AccessControlAllowOrigin:      {"*"},
					headers.AccessControlAllowCredentials: {"true"},
					headers.ContentType:                   {textPlain},
					headers.AccessControlAllowMethods: {
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
					headers.AccessControlAllowOrigin:      {"*"},
					headers.AccessControlAllowCredentials: {"true"},
					headers.ContentType:                   {textPlain},
					"X-Key":                               {"X-Key-Value"},
					headers.AccessControlAllowMethods: {
						"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
					},
				},
			},
			{
				name: "should override default headers",
				response: mock.Response{
					Code: 200,
					Headers: map[string]string{
						headers.AccessControlAllowOrigin:      "localhost",
						headers.AccessControlAllowCredentials: "false",
						headers.ContentType:                   "none",
					},
					RawContent: "test content",
				},
				expected: map[string][]string{
					headers.AccessControlAllowOrigin:      {"localhost"},
					headers.AccessControlAllowCredentials: {"false"},
					headers.ContentType:                   {"none"},
					headers.AccessControlAllowMethods: {
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
