package tui_test

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestPrintResponse(t *testing.T) {
	t.Run("should correctly format", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
		logger := tui.CreateLogger(log.Default(), styles.ProxyStyle.Render("Test"))

		tests := []struct {
			name       string
			response   *http.Response
			request    *contracts.Request
			statusCode int
		}{
			{
				name:       "response with status code 1xx",
				statusCode: 100,
				request:    request(http.MethodPost, "/api/info"),
			},
			{
				name:       "response with success code 2xx",
				statusCode: 200,
				request:    request(http.MethodGet, "/help"),
			},
			{
				name:       "response with redirect code 3xx",
				statusCode: 300,
				request:    request(http.MethodPatch, "/api/user"),
			},
			{
				name:       "response with user request error code 4xx",
				statusCode: 400,
				request:    request(http.MethodDelete, "/api/user/permission"),
			},
			{
				name:       "response with internal server error code 5xx",
				statusCode: 500,
				request:    request(http.MethodPost, "/"),
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				t.Run("should print single line", testutils.UniqOutput(output, func(t *testing.T, _ *bytes.Buffer) {
					tui.PrintResponse(logger, testCase.request, testCase.statusCode)

					assert.Equal(t, 1, lipgloss.Height(strings.Trim(output.String(), "\n")))
				}))

				t.Run("should print correctly", testutils.UniqOutput(output, func(t *testing.T, _ *bytes.Buffer) {
					tui.PrintResponse(logger, testCase.request, testCase.statusCode)

					snaps.MatchSnapshot(t, output.String())
				}))
			})
		}
	}))

	t.Run("should panic for status code less then 100", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
		logger := tui.CreateLogger(log.Default(), styles.ProxyStyle.Render("Test"))

		assert.Panics(t, func() {
			tui.PrintResponse(logger, request(http.MethodGet, "/"), 50)
		})
	}))

	t.Run("should panic for status code great then 599", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
		logger := tui.CreateLogger(log.Default(), styles.ProxyStyle.Render("Test"))

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
