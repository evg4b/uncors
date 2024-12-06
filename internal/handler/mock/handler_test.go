package mock_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/evg4b/uncors/pkg/fakedata"

	"github.com/charmbracelet/log"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testconstants"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

const (
	textPlain = "text/plain; charset=utf-8"
	imagePng  = "image/png"
)

const (
	textContent     = "status: ok"
	jsonContent     = `{ "test": "ok" }`
	htmlContent     = "<html></html>"
	pngContent      = "\x89PNG\x0D\x0A\x1A\x0A"
	fakeJSONContent = "{\"foo\":\"Yourselves that school smoothly next.\"}\n"
)

const (
	textFile = "test.txt"
	jsonFile = "test.json"
	htmlFile = "test.html"
	pngFile  = "test.png"
)

func TestHandler(t *testing.T) {
	fileSystem := testutils.FsFromMap(t, map[string]string{
		textFile: textContent,
		jsonFile: jsonContent,
		htmlFile: htmlContent,
		pngFile:  pngContent,
	})

	t.Run("mock content", func(t *testing.T) {
		tests := []struct {
			name     string
			response config.Response
			expected string
		}{
			{
				name:     "raw content",
				response: config.Response{Raw: jsonContent},
				expected: jsonContent,
			},
			{
				name:     "file content",
				response: config.Response{File: jsonFile},
				expected: jsonContent,
			},
			{
				name: "fake content",
				response: config.Response{
					Seed: 123,
					Fake: &fakedata.Node{
						Type: "object",
						Properties: map[string]fakedata.Node{
							"foo": {Type: "sentence"},
						},
					},
				},
				expected: fakeJSONContent,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := mock.NewMockHandler(
					mock.WithLogger(log.New(io.Discard)),
					mock.WithResponse(testCase.response),
					mock.WithFileSystem(fileSystem),
					mock.WithGenerator(fakedata.NewGoFakeItGenerator()),
					mock.WithAfter(func(_ time.Duration) <-chan time.Time {
						return time.After(time.Nanosecond)
					}),
				)

				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "/", nil)
				handler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

				body := testutils.ReadBody(t, recorder)
				assert.EqualValues(t, testCase.expected, body)
			})
		}
	})

	t.Run("content type detection", func(t *testing.T) {
		tests := []struct {
			name     string
			response config.Response
			expected string
		}{
			{
				name:     "raw content with plain text",
				response: config.Response{Raw: textContent},
				expected: textPlain,
			},
			{
				name:     "raw content with json",
				response: config.Response{Raw: jsonContent},
				expected: textPlain,
			},
			{
				name:     "raw content with html",
				response: config.Response{Raw: htmlContent},
				expected: "text/html; charset=utf-8",
			},
			{
				name:     "raw content with png",
				response: config.Response{Raw: pngContent},
				expected: imagePng,
			},
			{
				name:     "file with plain text",
				response: config.Response{File: textFile},
				expected: textPlain,
			},
			{
				name:     "file with json",
				response: config.Response{File: jsonFile},
				expected: "application/json",
			},
			{
				name:     "file with html",
				response: config.Response{File: htmlFile},
				expected: "text/html; charset=utf-8",
			},
			{
				name:     "file with png",
				response: config.Response{File: pngFile},
				expected: imagePng,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := mock.NewMockHandler(
					mock.WithLogger(log.New(io.Discard)),
					mock.WithResponse(testCase.response),
					mock.WithFileSystem(fileSystem),
					mock.WithAfter(func(_ time.Duration) <-chan time.Time {
						return time.After(time.Nanosecond)
					}),
				)

				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "/", nil)
				handler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

				header := testutils.ReadHeader(t, recorder)
				assert.EqualValues(t, testCase.expected, header.Get(headers.ContentType))
			})
		}
	})

	t.Run("headers settings", func(t *testing.T) {
		tests := []struct {
			name     string
			response config.Response
			expected http.Header
		}{
			{
				name: "should put default CORS headers",
				response: config.Response{
					Code: http.StatusOK,
					Raw:  textContent,
				},
				expected: map[string][]string{
					headers.AccessControlAllowOrigin:      {"*"},
					headers.AccessControlAllowCredentials: {"true"},
					headers.ContentType:                   {textPlain},
					headers.AccessControlAllowMethods:     {testconstants.AllMethods},
				},
			},
			{
				name: "should set response code",
				response: config.Response{
					Code: http.StatusOK,
					Raw:  textContent,
				},
				expected: map[string][]string{
					headers.AccessControlAllowOrigin:      {"*"},
					headers.AccessControlAllowCredentials: {"true"},
					headers.ContentType:                   {textPlain},
					headers.AccessControlAllowMethods:     {testconstants.AllMethods},
				},
			},
			{
				name: "should set custom headers",
				response: config.Response{
					Code: http.StatusOK,
					Headers: map[string]string{
						"X-Key": "X-Key-Value",
					},
					Raw: textContent,
				},
				expected: map[string][]string{
					headers.AccessControlAllowOrigin:      {"*"},
					headers.AccessControlAllowCredentials: {"true"},
					headers.ContentType:                   {textPlain},
					"X-Key":                               {"X-Key-Value"},
					headers.AccessControlAllowMethods:     {testconstants.AllMethods},
				},
			},
			{
				name: "should override default headers",
				response: config.Response{
					Code: http.StatusOK,
					Headers: map[string]string{
						headers.AccessControlAllowOrigin:      hosts.Localhost.Host(),
						headers.AccessControlAllowCredentials: "false",
						headers.ContentType:                   "none",
					},
					Raw: textContent,
				},
				expected: map[string][]string{
					headers.AccessControlAllowOrigin:      {hosts.Localhost.Host()},
					headers.AccessControlAllowCredentials: {"false"},
					headers.ContentType:                   {"none"},
					headers.AccessControlAllowMethods:     {testconstants.AllMethods},
				},
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := mock.NewMockHandler(
					mock.WithLogger(log.New(io.Discard)),
					mock.WithResponse(testCase.response),
					mock.WithFileSystem(fileSystem),
					mock.WithAfter(func(_ time.Duration) <-chan time.Time {
						return time.After(time.Nanosecond)
					}),
				)

				request := httptest.NewRequest(http.MethodGet, "/", nil)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

				assert.EqualValues(t, testCase.expected, testutils.ReadHeader(t, recorder))
				assert.Equal(t, http.StatusOK, recorder.Code)
			})
		}
	})

	t.Run("status code", func(t *testing.T) {
		const content = "test content"

		tests := []struct {
			name     string
			response config.Response
			expected int
		}{
			{
				name: "provide 201 code",
				response: config.Response{
					Code: http.StatusCreated,
					Raw:  content,
				},
				expected: http.StatusCreated,
			},
			{
				name: "provide 503 code",
				response: config.Response{
					Code: http.StatusServiceUnavailable,
					Raw:  content,
				},
				expected: http.StatusServiceUnavailable,
			},
			{
				name: "automatically provide 200 code",
				response: config.Response{
					Raw: content,
				},
				expected: http.StatusOK,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := mock.NewMockHandler(
					mock.WithLogger(log.New(io.Discard)),
					mock.WithResponse(testCase.response),
					mock.WithFileSystem(fileSystem),
					mock.WithAfter(func(_ time.Duration) <-chan time.Time {
						return time.After(time.Nanosecond)
					}),
				)

				request := httptest.NewRequest(http.MethodGet, "/", nil)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

				assert.Equal(t, testCase.expected, recorder.Code)
			})
		}
	})

	t.Run("mock response delay", func(t *testing.T) {
		t.Run("correctly handle delay", func(t *testing.T) {
			tests := []struct {
				name           string
				response       config.Response
				shouldBeCalled bool
				expected       time.Duration
			}{
				{
					name: "3s delay",
					response: config.Response{
						Code:  http.StatusCreated,
						Delay: 3 * time.Second,
					},
					shouldBeCalled: true,
					expected:       3 * time.Second,
				},
				{
					name: "15h delay",
					response: config.Response{
						Code:  http.StatusCreated,
						Delay: 15 * time.Hour,
					},
					shouldBeCalled: true,
					expected:       15 * time.Hour,
				},
				{
					name: "0s delay",
					response: config.Response{
						Code:  http.StatusCreated,
						Delay: 0 * time.Second,
					},
					shouldBeCalled: false,
				},
				{
					name: "delay is not set",
					response: config.Response{
						Code: http.StatusCreated,
					},
					shouldBeCalled: false,
				},
				{
					name: "incorrect delay",
					response: config.Response{
						Code:  http.StatusCreated,
						Delay: -13 * time.Minute,
					},
					shouldBeCalled: false,
				},
			}
			for _, testCase := range tests {
				t.Run(testCase.name, func(t *testing.T) {
					called := false
					handler := mock.NewMockHandler(
						mock.WithLogger(log.New(io.Discard)),
						mock.WithResponse(testCase.response),
						mock.WithFileSystem(fileSystem),
						mock.WithAfter(func(duration time.Duration) <-chan time.Time {
							assert.Equal(t, testCase.expected, duration)
							called = true

							return time.After(time.Nanosecond)
						}),
					)

					request := httptest.NewRequest(http.MethodGet, "/", nil)
					recorder := httptest.NewRecorder()

					handler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

					assert.Equal(t, called, testCase.shouldBeCalled)
				})
			}
		})

		t.Run("correctly cancel delay", func(t *testing.T) {
			handler := mock.NewMockHandler(
				mock.WithLogger(log.New(io.Discard)),
				mock.WithResponse(config.Response{
					Code:  http.StatusOK,
					Delay: 1 * time.Hour,
					Raw:   "Text content",
				}),
				mock.WithFileSystem(fileSystem),
				mock.WithAfter(time.After),
			)

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			ctx, cancel := context.WithCancel(context.Background())
			recorder := httptest.NewRecorder()

			var waitGroup sync.WaitGroup
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				handler.ServeHTTP(contracts.WrapResponseWriter(recorder), request.WithContext(ctx))
			}()

			cancel()

			waitGroup.Wait()

			assert.Equal(t, "", testutils.ReadBody(t, recorder))
			assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
		})
	})
}
