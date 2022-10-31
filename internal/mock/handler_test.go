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

	t.Run("set correct content type", func(t *testing.T) {
		tests := []struct {
			name   string
			body   string
			expted string
		}{
			{
				name:   "plain text",
				body:   `status: ok`,
				expted: "text/plain; charset=utf-8",
			},
			{
				name:   "json",
				body:   `{ "status": "ok" }`,
				expted: "text/plain; charset=utf-8",
			},
			{
				name:   "html",
				body:   `<html></html>`,
				expted: "text/html; charset=utf-8",
			},
			{
				name:   "xml",
				body:   `<?xml />`,
				expted: "text/xml; charset=utf-8",
			},
			{
				name:   "png",
				body:   "\x89PNG\x0D\x0A\x1A\x0A",
				expted: "image/png",
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := mock.NewMockHandler(mock.WithLogger(logger), mock.WithMock(mock.Mock{
					Path: "/",
					Response: mock.Response{
						Code:       200,
						RawContent: testCase.body,
					},
				}))

				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "/", nil)
				handler.ServeHTTP(recorder, request)

				header := testutils.ReadHeader(t, recorder)
				assert.EqualValues(t, testCase.expted, header.Get("Content-Type"))
			})
		}
	})

	tests := []struct {
		name   string
		mock   mock.Mock
		body   string
		header http.Header
		code   int
	}{
		{
			name: "should put default CORS headers",
			mock: mock.Mock{
				Path: "/",
				Response: mock.Response{
					Code:       200,
					RawContent: "test content",
				},
			},
			body: "test content",
			header: map[string][]string{
				"Access-Control-Allow-Origin":      {"*"},
				"Access-Control-Allow-Credentials": {"true"},
				"Content-Type":                     {"text/plain; charset=utf-8"},
				"Access-Control-Allow-Methods": {
					"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
				},
			},
			code: 200,
		},
		{
			name: "should set response code",
			mock: mock.Mock{
				Path: "/",
				Response: mock.Response{
					Code:       345,
					RawContent: "test content",
				},
			},
			body: "test content",
			header: map[string][]string{
				"Access-Control-Allow-Origin":      {"*"},
				"Access-Control-Allow-Credentials": {"true"},
				"Content-Type":                     {"text/plain; charset=utf-8"},
				"Access-Control-Allow-Methods": {
					"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
				},
			},
			code: 345,
		},
		{
			name: "should set custom headers",
			mock: mock.Mock{
				Path: "/",
				Response: mock.Response{
					Code:       345,
					Headers:    map[string]string{},
					RawContent: "test content",
				},
			},
			body: "test content",
			header: map[string][]string{
				"Access-Control-Allow-Origin":      {"*"},
				"Access-Control-Allow-Credentials": {"true"},
				"Content-Type":                     {"text/plain; charset=utf-8"},
				"Access-Control-Allow-Methods": {
					"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
				},
			},
			code: 345,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			handler := mock.NewMockHandler(
				mock.WithMock(testCase.mock),
				mock.WithLogger(logger),
			)

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assert.EqualValues(t, testCase.header, testutils.ReadHeader(t, recorder))
			assert.EqualValues(t, testCase.body, testutils.ReadBody(t, recorder))
			assert.Equal(t, testCase.code, recorder.Code)
		})
	}
}
