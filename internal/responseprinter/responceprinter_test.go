// nolint: bodyclose, lll
package responseprinter_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/evg4b/uncors/internal/responseprinter"
	"github.com/stretchr/testify/assert"
)

func makeResponseURL(code int, method string, path string) *http.Response {
	return &http.Response{
		Request: &http.Request{
			Method: method,
			URL: &url.URL{
				Scheme: "https",
				Host:   "api.domain.com",
				Path:   path,
			},
		},
		StatusCode: code,
	}
}

func TestPrintResponse(t *testing.T) {
	t.Run("should correctly format", func(t *testing.T) {
		tests := []struct {
			name     string
			response *http.Response
			expected string
		}{
			{
				name:     "information response with status code 1xx",
				response: makeResponseURL(100, http.MethodPost, "/api/info"),
				expected: "\x1b[30;46m\x1b[30;46m 100 POST \x1b[0m\x1b[0m \x1b[96m\x1b[96mhttps://api.domain.com/api/info\x1b[0m\x1b[0m",
			},
			{
				name:     "information response with success code 2xx",
				response: makeResponseURL(200, http.MethodGet, "/help"),
				expected: "\x1b[30;42m\x1b[30;42m 200 GET \x1b[0m\x1b[0m \x1b[32m\x1b[32mhttps://api.domain.com/help\x1b[0m\x1b[0m",
			},
			{
				name:     "information response with redirect code 3xx",
				response: makeResponseURL(300, http.MethodPatch, "/api/user"),
				expected: "\x1b[30;43m\x1b[30;43m 300 PATCH \x1b[0m\x1b[0m \x1b[33m\x1b[33mhttps://api.domain.com/api/user\x1b[0m\x1b[0m",
			},
			{
				name:     "information response with user request error code 4xx",
				response: makeResponseURL(400, http.MethodDelete, "/api/user/permission"),
				expected: "\x1b[30;101m\x1b[30;101m 400 DELETE \x1b[0m\x1b[0m \x1b[91m\x1b[91mhttps://api.domain.com/api/user/permission\x1b[0m\x1b[0m",
			},
			{
				name:     "information response with internal server error code 5xx",
				response: makeResponseURL(500, http.MethodPost, "/"),
				expected: "\x1b[30;101m\x1b[30;101m 500 POST \x1b[0m\x1b[0m \x1b[91m\x1b[91mhttps://api.domain.com/\x1b[0m\x1b[0m",
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				actual := responseprinter.PrintResponse(testCase.response)

				assert.Equal(t, testCase.expected, actual)
			})
		}
	})

	t.Run("should panic for status code less then 100", func(t *testing.T) {
		assert.Panics(t, func() {
			responseprinter.PrintResponse(makeResponseURL(50, http.MethodGet, "/"))
		})
	})

	t.Run("should panic for status code great then 599", func(t *testing.T) {
		assert.Panics(t, func() {
			responseprinter.PrintResponse(makeResponseURL(600, http.MethodGet, "/"))
		})
	})
}
