package responceprinter

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeRospoceUrl(code int, method string, path string) *http.Response {
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

func TestPrintResponce(t *testing.T) {
	t.Run("should correctly format", func(t *testing.T) {
		tests := []struct {
			name     string
			responce *http.Response
			expected string
		}{
			{
				name:     "information respoce with status code 1xx",
				responce: makeRospoceUrl(100, "POST", "/api/info"),
				expected: "\x1b[30;46m\x1b[30;46m 100 POST \x1b[0m\x1b[0m \x1b[96m\x1b[96mhttps://api.domain.com/api/info\x1b[0m\x1b[0m",
			},
			{
				name:     "information respoce with success code 2xx",
				responce: makeRospoceUrl(200, "GET", "/help"),
				expected: "\x1b[30;42m\x1b[30;42m 200 GET \x1b[0m\x1b[0m \x1b[32m\x1b[32mhttps://api.domain.com/help\x1b[0m\x1b[0m",
			},
			{
				name:     "information respoce with redirect code 3xx",
				responce: makeRospoceUrl(300, "PATCH", "/api/user"),
				expected: "\x1b[30;43m\x1b[30;43m 300 PATCH \x1b[0m\x1b[0m \x1b[33m\x1b[33mhttps://api.domain.com/api/user\x1b[0m\x1b[0m",
			},
			{
				name:     "information respoce with user request error code 4xx",
				responce: makeRospoceUrl(400, "DELETE", "/api/user/repmission"),
				expected: "\x1b[30;101m\x1b[30;101m 400 DELETE \x1b[0m\x1b[0m \x1b[91m\x1b[91mhttps://api.domain.com/api/user/repmission\x1b[0m\x1b[0m",
			},
			{
				name:     "information respoce with internal server error code 5xx",
				responce: makeRospoceUrl(500, "POST", "/"),
				expected: "\x1b[30;101m\x1b[30;101m 500 POST \x1b[0m\x1b[0m \x1b[91m\x1b[91mhttps://api.domain.com/\x1b[0m\x1b[0m",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				actual := PrintResponce(tt.responce)

				assert.Equal(t, tt.expected, actual)
			})
		}
	})

	t.Run("should panic for status code less then 100", func(t *testing.T) {
		assert.Panics(t, func() {
			PrintResponce(makeRospoceUrl(50, "GET", "/"))
		})
	})

	t.Run("should panic for status codegreat then 599", func(t *testing.T) {
		assert.Panics(t, func() {
			PrintResponce(makeRospoceUrl(600, "GET", "/"))
		})
	})
}
