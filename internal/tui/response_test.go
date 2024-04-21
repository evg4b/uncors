package tui_test

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestPrintResponse(t *testing.T) {
	t.Run("should correctly format", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
		logger := styles.CreateLogger(log.Default(), styles.ProxyStyle.Render("Test"))

		tests := []struct {
			name       string
			response   *http.Response
			request    *contracts.Request
			statusCode int
			expected   string
		}{
			{
				name:       "response with status code 1xx",
				statusCode: 100,
				request:    request(http.MethodPost, "/api/info"),
				expected: "\x1b[48;2;105;113;247m \x1b[0m\x1b[38;2;0;0;0;48;2;105;" +
					"113;247mTest\x1b[0m\x1b[48;2;105;113;247m \x1b[0m\x1b[48;2;105;113;" +
					"247m  \x1b[0m \x1b[48;2;0;113;206m \x1b[0m\x1b[38;2;0;0;0;48;2;0;113;" +
					"206m100 POST\x1b[0m\x1b[48;2;0;113;206m \x1b[0m \x1b[38;2;0;113;" +
					"206mhttps://api.domain.com/api/info\x1b[0m\n",
			},
			{
				name:       "response with success code 2xx",
				statusCode: 200,
				request:    request(http.MethodGet, "/help"),
				expected: "\x1b[48;2;105;113;247m \x1b[0m\x1b[38;2;0;0;0;48;2;105;113;" +
					"247mTest\x1b[0m\x1b[48;2;105;113;247m \x1b[0m\x1b[48;2;105;113;" +
					"247m  \x1b[0m \x1b[48;2;0;168;107m \x1b[0m\x1b[38;2;0;0;0;48;2;0;" +
					"168;107m200 GET\x1b[0m\x1b[48;2;0;168;107m \x1b[0m\x1b[48;2;0;168;" +
					"107m \x1b[0m \x1b[38;2;0;168;107mhttps://api.domain.com/help\x1b[0m\n",
			},
			{
				name:       "response with redirect code 3xx",
				statusCode: 300,
				request:    request(http.MethodPatch, "/api/user"),
				expected: "\x1b[48;2;105;113;247m \x1b[0m\x1b[38;2;0;0;0;48;2;105;113" +
					";247mTest\x1b[0m\x1b[48;2;105;113;247m \x1b[0m\x1b[48;2;105;113;" +
					"247m  \x1b[0m \x1b[48;2;255;211;0m \x1b[0m\x1b[38;2;0;0;0;48;2;255;" +
					"211;0m300\x1b[0m\x1b[48;2;255;211;0m \x1b[0m\x1b[48;2;255;211;" +
					"0m     \x1b[0m\n\x1b[48;2;255;211;0m \x1b[0m\x1b[38;2;0;0;0;48;2;255;" +
					"211;0mPATCH\x1b[0m\x1b[48;2;255;211;0m \x1b[0m\x1b[48;2;255;211;" +
					"0m   \x1b[0m \x1b[38;2;255;211;0mhttps://api.domain.com/api/user\x1b[0m\n",
			},
			{
				name:       "response with user request error code 4xx",
				statusCode: 400,
				request:    request(http.MethodDelete, "/api/user/permission"),
				expected: "\x1b[48;2;105;113;247m \x1b[0m\x1b[38;2;0;0;0;48;2;105;" +
					"113;247mTest\x1b[0m\x1b[48;2;105;113;247m \x1b[0m\x1b[48;2;105;" +
					"113;247m  \x1b[0m \x1b[48;2;220;1;0m \x1b[0m\x1b[38;2;0;0;0;48;2;" +
					"220;1;0m400\x1b[0m\x1b[48;2;220;1;0m \x1b[0m\x1b[48;2;220;1;" +
					"0m     \x1b[0m\n\x1b[48;2;220;1;0m \x1b[0m\x1b[38;2;0;0;0;48;2;" +
					"220;1;0mDELETE\x1b[0m\x1b[48;2;220;1;0m \x1b[0m\x1b[48;2;220;1;" +
					"0m  \x1b[0m \x1b[38;2;220;1;0mhttps://api.domain.com/api/user/permission\x1b[0m\n",
			},
			{
				name:       "response with internal server error code 5xx",
				statusCode: 500,
				request:    request(http.MethodPost, "/"),
				expected: "\x1b[48;2;105;113;247m \x1b[0m\x1b[38;2;0;0;0;48;2;105;113;" +
					"247mTest\x1b[0m\x1b[48;2;105;113;247m \x1b[0m\x1b[48;2;105;113;" +
					"247m  \x1b[0m \x1b[48;2;220;1;0m \x1b[0m\x1b[38;2;0;0;0;48;2;220;" +
					"1;0m500 POST\x1b[0m\x1b[48;2;220;1;0m \x1b[0m \x1b[38;2;220;1;" +
					"0mhttps://api.domain.com/\x1b[0m\n",
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, testutils.UniqOutput(output, func(t *testing.T, output *bytes.Buffer) {
				tui.PrintResponse(logger, testCase.request, testCase.statusCode)

				assert.Equal(t, testCase.expected, output.String())
			}))
		}
	}))

	t.Run("should panic for status code less then 100", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
		logger := styles.CreateLogger(log.Default(), styles.ProxyStyle.Render("Test"))

		assert.Panics(t, func() {
			tui.PrintResponse(logger, request(http.MethodGet, "/"), 50)
		})
	}))

	t.Run("should panic for status code great then 599", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
		logger := styles.CreateLogger(log.Default(), styles.ProxyStyle.Render("Test"))

		assert.Panics(t, func() {
			tui.PrintResponse(logger, request(http.MethodGet, "/"), 600)
		})
	}))
}

func request(method string, path string) *http.Request {
	return &http.Request{
		Method: method,
		URL: &url.URL{
			Scheme: "https",
			Host:   "api.domain.com",
			Path:   path,
		},
	}
}
