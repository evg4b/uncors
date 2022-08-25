// nolint: bodyclose, lll
package responseprinter_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/evg4b/uncors/internal/responseprinter"
	"github.com/stretchr/testify/assert"
)

func makeRospoceURL(code int, method string, path string) *http.Response {
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

func TestPrintresponse(t *testing.T) {
	t.Run("should correctly format", func(t *testing.T) {
		tests := []struct {
			name     string
			response *http.Response
			expected string
		}{
			{
				name:     "information respoce with status code 1xx",
				response: makeRospoceURL(100, http.MethodPost, "/api/info"),
				expected: "\x1b[30;46m\x1b[30;46m 100 POST \x1b[0m\x1b[0m \x1b[96m\x1b[96mhttps://api.domain.com/api/info\x1b[0m\x1b[0m",
			},
			{
				name:     "information respoce with success code 2xx",
				response: makeRospoceURL(200, http.MethodGet, "/help"),
				expected: "\x1b[30;42m\x1b[30;42m 200 GET \x1b[0m\x1b[0m \x1b[32m\x1b[32mhttps://api.domain.com/help\x1b[0m\x1b[0m",
			},
			{
				name:     "information respoce with redirect code 3xx",
				response: makeRospoceURL(300, http.MethodPatch, "/api/user"),
				expected: "\x1b[30;43m\x1b[30;43m 300 PATCH \x1b[0m\x1b[0m \x1b[33m\x1b[33mhttps://api.domain.com/api/user\x1b[0m\x1b[0m",
			},
			{
				name:     "information respoce with user request error code 4xx",
				response: makeRospoceURL(400, http.MethodDelete, "/api/user/repmission"),
				expected: "\x1b[30;101m\x1b[30;101m 400 DELETE \x1b[0m\x1b[0m \x1b[91m\x1b[91mhttps://api.domain.com/api/user/repmission\x1b[0m\x1b[0m",
			},
			{
				name:     "information respoce with internal server error code 5xx",
				response: makeRospoceURL(500, http.MethodPost, "/"),
				expected: "\x1b[30;101m\x1b[30;101m 500 POST \x1b[0m\x1b[0m \x1b[91m\x1b[91mhttps://api.domain.com/\x1b[0m\x1b[0m",
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				actual := responseprinter.Printresponse(testCase.response)

				assert.Equal(t, testCase.expected, actual)
			})
		}
	})

	t.Run("should panic for status code less then 100", func(t *testing.T) {
		assert.Panics(t, func() {
			responseprinter.Printresponse(makeRospoceURL(50, http.MethodGet, "/"))
		})
	})

	t.Run("should panic for status codegreat then 599", func(t *testing.T) {
		assert.Panics(t, func() {
			responseprinter.Printresponse(makeRospoceURL(600, http.MethodGet, "/"))
		})
	})
}
