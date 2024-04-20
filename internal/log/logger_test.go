// nolint: bodyclose, goconst
package log_test

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/log"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
)

const (
	testMessage  = "this is test message"
	testFMessage = "this is %s message"
	prefix       = " Test "
)

func TestPrefixedLogger(t *testing.T) {
	log.EnableOutput()
	log.DisableColor()
	log.EnableDebugMessages()

	t.Run("prefix printing", func(t *testing.T) {
		logger := log.NewLogger(prefix)

		t.Run("Error", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			logger.Error(testMessage)

			assert.Equal(t, "  Test      ERROR  this is test message\n", output.String())
		}))

		t.Run("Errorf", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			logger.Errorf(testFMessage, "Error")

			assert.Equal(t, "  Test      ERROR  this is Error message\n", output.String())
		}))

		t.Run("Info", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			logger.Info(testMessage)

			assert.Equal(t, "  Test       INFO  this is test message\n", output.String())
		}))

		t.Run("Infof", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			logger.Infof(testFMessage, "Info")

			assert.Equal(t, "  Test       INFO  this is Info message\n", output.String())
		}))

		t.Run("Warning", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			logger.Warning(testMessage)

			assert.Equal(t, "  Test    WARNING  this is test message\n", output.String())
		}))

		t.Run("Warningf", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			logger.Warningf(testFMessage, "Warning")

			assert.Equal(t, "  Test    WARNING  this is Warning message\n", output.String())
		}))

		t.Run("Debug", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			logger.Debug(testMessage)

			assert.Equal(t, "  Test      DEBUG  this is test message\n", output.String())
		}))

		t.Run("Debugf", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			logger.Debugf(testFMessage, "Debug")

			assert.Equal(t, "  Test      DEBUG  this is Debug message\n", output.String())
		}))
	})

	t.Run("custom output", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
		buffer := &bytes.Buffer{}

		logger := log.NewLogger(prefix, log.WithOutput(buffer))

		logger.Info("Test message")

		assert.Empty(t, output.String())
		assert.Equal(t, "  Test       INFO  Test message\n", buffer.String())
	}))

	t.Run("custom styles", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
		log.EnableColor()

		logger := log.NewLogger(prefix, log.WithStyle(&pterm.Style{
			pterm.BgBlue,
			pterm.BgLightBlue,
		}))

		logger.Info("Test message")

		expected := "\x1b[44;104m\x1b[44;104m  Test  \x1b[0m\x1b[0m \x1b[39;49m\x1b[39;49m\x1b[30;46m\x1b[30;46m" +
			"    INFO \x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m \x1b[96m\x1b[96mTest message\x1b[0m\x1b[39;49m\x1b[0m" +
			"\x1b[39;49m\x1b[0m\x1b[0m\n"

		assert.Equal(t, expected, output.String())
	}))

	t.Run("printResponse", func(t *testing.T) {
		log.EnableColor()
		logger := log.NewLogger(prefix, log.WithStyle(&pterm.Style{pterm.FgBlack, pterm.BgLightBlue}))

		t.Run("should correctly format", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
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
					expected: "\x1b[30;104m\x1b[30;104m  Test  \x1b[0m\x1b[0m \x1b[39;49m\x1b[39;49m\x1b[30;46m" +
						"\x1b[30;46m 100 POST \x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m \x1b[96m\x1b[96mhttps://api.domain" +
						".com/api/info\x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m\x1b[0m\x1b[0m\n",
				},
				{
					name:       "response with success code 2xx",
					statusCode: 200,
					request:    request(http.MethodGet, "/help"),
					expected: "\x1b[30;104m\x1b[30;104m  Test  \x1b[0m\x1b[0m \x1b[39;49m\x1b[39;49m\x1b[30;46m" +
						"\x1b[30;46m 100 POST \x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m \x1b[96m\x1b[96mhttps://api." +
						"domain.com/api/info\x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m\x1b[0m\x1b[0m\n\x1b[30;104m\x1b" +
						"[30;104m  Test  \x1b[0m\x1b[0m \x1b[39;49m\x1b[39;49m\x1b[30;42m\x1b[30;42m 200 GET \x1b" +
						"[0m\x1b[39;49m\x1b[0m\x1b[39;49m \x1b[32m\x1b[32mhttps://api.domain.com/help\x1b[0m\x1b" +
						"[39;49m\x1b[0m\x1b[39;49m\x1b[0m\x1b[0m\n",
				},
				{
					name:       "response with redirect code 3xx",
					statusCode: 300,
					request:    request(http.MethodPatch, "/api/user"),
					expected: "\x1b[30;104m\x1b[30;104m  Test  \x1b[0m\x1b[0m \x1b[39;49m\x1b[39;49m\x1b[30;46m\x1b" +
						"[30;46m 100 POST \x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m \x1b[96m\x1b[96mhttps://api.domain" +
						".com/api/info\x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m\x1b[0m\x1b[0m\n\x1b[30;104m\x1b[30;104m  " +
						"Test  \x1b[0m\x1b[0m \x1b[39;49m\x1b[39;49m\x1b[30;42m\x1b[30;42m 200 GET \x1b[0m\x1b[39;" +
						"49m\x1b[0m\x1b[39;49m \x1b[32m\x1b[32mhttps://api.domain.com/help\x1b[0m\x1b[39;49m\x1b[0m" +
						"\x1b[39;49m\x1b[0m\x1b[0m\n\x1b[30;104m\x1b[30;104m  Test  \x1b[0m\x1b[0m \x1b[39;49m\x1b" +
						"[39;49m\x1b[30;43m\x1b[30;43m 300 PATCH \x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m \x1b[33m\x1b" +
						"[33mhttps://api.domain.com/api/user\x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m\x1b[0m\x1b[0m\n",
				},
				{
					name:       "response with user request error code 4xx",
					statusCode: 400,
					request:    request(http.MethodDelete, "/api/user/permission"),
					expected: "\x1b[30;104m\x1b[30;104m  Test  \x1b[0m\x1b[0m \x1b[39;49m\x1b[39;49m\x1b[30;46m" +
						"\x1b[30;46m 100 POST \x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m \x1b[96m\x1b[96mhttps://api." +
						"domain.com/api/info\x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m\x1b[0m\x1b[0m\n\x1b[30;104m\x1b" +
						"[30;104m  Test  \x1b[0m\x1b[0m \x1b[39;49m\x1b[39;49m\x1b[30;42m\x1b[30;42m 200 GET \x1b" +
						"[0m\x1b[39;49m\x1b[0m\x1b[39;49m \x1b[32m\x1b[32mhttps://api.domain.com/help\x1b[0m\x1b" +
						"[39;49m\x1b[0m\x1b[39;49m\x1b[0m\x1b[0m\n\x1b[30;104m\x1b[30;104m  Test  \x1b[0m\x1b[0m \x1b" +
						"[39;49m\x1b[39;49m\x1b[30;43m\x1b[30;43m 300 PATCH \x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m " +
						"\x1b[33m\x1b[33mhttps://api.domain.com/api/user\x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m\x1b" +
						"[0m\x1b[0m\n\x1b[30;104m\x1b[30;104m  Test  \x1b[0m\x1b[0m \x1b[39;49m\x1b[39;49m\x1b" +
						"[30;101m\x1b[30;101m 400 DELETE \x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m \x1b[91m\x1b" +
						"[91mhttps://api.domain.com/api/user/permission\x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m" +
						"\x1b[0m\x1b[0m\n",
				},
				{
					name:       "response with internal server error code 5xx",
					statusCode: 500,
					request:    request(http.MethodPost, "/"),
					expected: "\x1b[30;104m\x1b[30;104m  Test  \x1b[0m\x1b[0m \x1b[39;49m\x1b[39;49m\x1b[30;46m\x1b" +
						"[30;46m 100 POST \x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m \x1b[96m\x1b[96mhttps://api.domain." +
						"com/api/info\x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m\x1b[0m\x1b[0m\n\x1b[30;104m\x1b[30;104m" +
						"  Test  \x1b[0m\x1b[0m \x1b[39;49m\x1b[39;49m\x1b[30;42m\x1b[30;42m 200 GET \x1b[0m" +
						"\x1b[39;49m\x1b[0m\x1b[39;49m \x1b[32m\x1b[32mhttps://api.domain.com/help\x1b[0m\x1b[39;" +
						"49m\x1b[0m\x1b[39;49m\x1b[0m\x1b[0m\n\x1b[30;104m\x1b[30;104m  Test  \x1b[0m\x1b[0m \x1b" +
						"[39;49m\x1b[39;49m\x1b[30;43m\x1b[30;43m 300 PATCH \x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m " +
						"\x1b[33m\x1b[33mhttps://api.domain.com/api/user\x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m\x1b" +
						"[0m\x1b[0m\n\x1b[30;104m\x1b[30;104m  Test  \x1b[0m\x1b[0m \x1b[39;49m\x1b[39;49m\x1b" +
						"[30;101m\x1b[30;101m 400 DELETE \x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m \x1b[91m\x1b" +
						"[91mhttps://api.domain.com/api/user/permission\x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m\x1b" +
						"[0m\x1b[0m\n\x1b[30;104m\x1b[30;104m  Test  \x1b[0m\x1b[0m \x1b[39;49m\x1b[39;49m\x1b" +
						"[30;101m\x1b[30;101m 500 POST \x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m \x1b[91m\x1b" +
						"[91mhttps://api.domain.com/\x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m\x1b[0m\x1b[0m\n",
				},
			}
			for _, testCase := range tests {
				t.Run(testCase.name, func(t *testing.T) {
					logger.PrintResponse(testCase.request, testCase.statusCode)

					assert.Equal(t, testCase.expected, output.String())
				})
			}
		}))

		t.Run("should panic for status code less then 100", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
			assert.Panics(t, func() {
				logger.PrintResponse(request(http.MethodGet, "/"), 50)
			})
		}))

		t.Run("should panic for status code great then 599", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
			assert.Panics(t, func() {
				logger.PrintResponse(request(http.MethodGet, "/"), 600)
			})
		}))
	})
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
