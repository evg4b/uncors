package mock

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

const textPlain = "text/plain; charset=utf-8"

const testContent = "test content"

func TestHandler(t *testing.T) {
	fileSystem := afero.NewMemMapFs()

	var makeHandler = func(t *testing.T, response Response) *internalHandler {
		return &internalHandler{
			logger:   mocks.NewNoopLogger(t),
			response: response,
			fs:       fileSystem,
		}
	}

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
				handler := makeHandler(t, Response{
					Code:       200,
					RawContent: testCase.body,
				})

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
			response Response
			expected http.Header
		}{
			{
				name: "should put default CORS headers",
				response: Response{
					Code:       200,
					RawContent: testContent,
				},
				expected: map[string][]string{
					headers.AccessControlAllowOrigin:      {"*"},
					headers.AccessControlAllowCredentials: {"true"},
					headers.ContentType:                   {textPlain},
					headers.AccessControlAllowMethods:     {mocks.AllMethods},
				},
			},
			{
				name: "should set response code",
				response: Response{
					Code:       200,
					RawContent: testContent,
				},
				expected: map[string][]string{
					headers.AccessControlAllowOrigin:      {"*"},
					headers.AccessControlAllowCredentials: {"true"},
					headers.ContentType:                   {textPlain},
					headers.AccessControlAllowMethods:     {mocks.AllMethods},
				},
			},
			{
				name: "should set custom headers",
				response: Response{
					Code: 200,
					Headers: map[string]string{
						"X-Key": "X-Key-Value",
					},
					RawContent: testContent,
				},
				expected: map[string][]string{
					headers.AccessControlAllowOrigin:      {"*"},
					headers.AccessControlAllowCredentials: {"true"},
					headers.ContentType:                   {textPlain},
					"X-Key":                               {"X-Key-Value"},
					headers.AccessControlAllowMethods:     {mocks.AllMethods},
				},
			},
			{
				name: "should override default headers",
				response: Response{
					Code: 200,
					Headers: map[string]string{
						headers.AccessControlAllowOrigin:      "localhost",
						headers.AccessControlAllowCredentials: "false",
						headers.ContentType:                   "none",
					},
					RawContent: testContent,
				},
				expected: map[string][]string{
					headers.AccessControlAllowOrigin:      {"localhost"},
					headers.AccessControlAllowCredentials: {"false"},
					headers.ContentType:                   {"none"},
					headers.AccessControlAllowMethods:     {mocks.AllMethods},
				},
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := makeHandler(t, testCase.response)

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
			response Response
			expected int
		}{
			{
				name: "provide 201 code",
				response: Response{
					Code: 201,
				},
				expected: 201,
			},
			{
				name: "provide 503 code",
				response: Response{
					Code: 503,
				},
				expected: 503,
			},
			{
				name:     "automatically provide 200 code",
				response: Response{},
				expected: 200,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := makeHandler(t, testCase.response)

				request := httptest.NewRequest(http.MethodGet, "/", nil)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)

				assert.Equal(t, testCase.expected, recorder.Code)
			})
		}
	})
}
