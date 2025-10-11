package options_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/options"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

func TestMiddleware(t *testing.T) {
	loggerMock := log.New(io.Discard)

	t.Run("for OPTIONS request", func(t *testing.T) {
		mockedNextHandler := mocks.FailNowHandlerMock(t)

		defaultControlAllowOrigin := []string{"*"}
		defaultControlAllowMethods := []string{
			"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
		}
		defaultControlAllowCredentials := []string{"true"}
		defaultControlAllowHeaders := []string{"*"}
		defaultControlMaxAge := []string{"86400"}
		defaultControlExposeHeaders := []string{"*"}

		defaultCorsHeaders := http.Header{
			headers.AccessControlAllowOrigin:      defaultControlAllowOrigin,
			headers.AccessControlAllowCredentials: defaultControlAllowCredentials,
			headers.AccessControlAllowHeaders:     defaultControlAllowHeaders,
			headers.AccessControlAllowMethods:     defaultControlAllowMethods,
			headers.AccessControlMaxAge:           defaultControlMaxAge,
			headers.AccessControlExposeHeaders:    defaultControlExposeHeaders,
		}

		type args struct {
			headers        map[string]string
			code           int
			requestHeaders http.Header
		}

		type expected struct {
			headers http.Header
			code    int
		}

		cases := []struct {
			name     string
			args     args
			expected expected
		}{
			{
				name: "default",
				args: args{},
				expected: expected{
					headers: defaultCorsHeaders,
					code:    http.StatusOK,
				},
			},
			{
				name: "with requested headers",
				args: args{
					requestHeaders: http.Header{
						headers.AccessControlRequestHeaders: []string{"Content-Type, Authorization"},
					},
				},
				expected: expected{
					headers: http.Header{
						headers.AccessControlAllowOrigin:      defaultControlAllowOrigin,
						headers.AccessControlAllowCredentials: defaultControlAllowCredentials,
						headers.AccessControlAllowHeaders:     []string{"Content-Type, Authorization"},
						headers.AccessControlAllowMethods:     defaultControlAllowMethods,
						headers.AccessControlMaxAge:           defaultControlMaxAge,
						headers.AccessControlExposeHeaders:    defaultControlExposeHeaders,
					},
					code: http.StatusOK,
				},
			},
			{
				name: "with requested method",
				args: args{
					requestHeaders: http.Header{
						headers.AccessControlRequestMethod: []string{"POST"},
					},
				},
				expected: expected{
					headers: http.Header{
						headers.AccessControlAllowOrigin:      defaultControlAllowOrigin,
						headers.AccessControlAllowCredentials: defaultControlAllowCredentials,
						headers.AccessControlAllowHeaders:     defaultControlAllowHeaders,
						headers.AccessControlAllowMethods:     []string{"POST"},
						headers.AccessControlMaxAge:           defaultControlMaxAge,
						headers.AccessControlExposeHeaders:    defaultControlExposeHeaders,
					},
					code: http.StatusOK,
				},
			},
			{
				name: "with requested headers and method",
				args: args{
					requestHeaders: http.Header{
						headers.AccessControlRequestHeaders: []string{"Content-Type, Authorization"},
						headers.AccessControlRequestMethod:  []string{"PUT"},
					},
				},
				expected: expected{
					headers: http.Header{
						headers.AccessControlAllowOrigin:      defaultControlAllowOrigin,
						headers.AccessControlAllowCredentials: defaultControlAllowCredentials,
						headers.AccessControlAllowHeaders:     []string{"Content-Type, Authorization"},
						headers.AccessControlAllowMethods:     []string{"PUT"},
						headers.AccessControlMaxAge:           defaultControlMaxAge,
						headers.AccessControlExposeHeaders:    defaultControlExposeHeaders,
					},
					code: http.StatusOK,
				},
			},
			{
				name: "custom headers",
				args: args{
					headers: map[string]string{
						headers.AcceptLanguage: "en",
					},
				},
				expected: expected{
					headers: http.Header{
						headers.AccessControlAllowOrigin:      defaultControlAllowOrigin,
						headers.AccessControlAllowCredentials: defaultControlAllowCredentials,
						headers.AccessControlAllowHeaders:     defaultControlAllowHeaders,
						headers.AccessControlAllowMethods:     defaultControlAllowMethods,
						headers.AccessControlMaxAge:           defaultControlMaxAge,
						headers.AccessControlExposeHeaders:    defaultControlExposeHeaders,
						headers.AcceptLanguage:                []string{"en"},
					},
					code: http.StatusOK,
				},
			},
			{
				name: "custom headers with override",
				args: args{
					headers: map[string]string{
						headers.AcceptLanguage:           "en",
						headers.AccessControlAllowOrigin: "https://example.com",
					},
				},
				expected: expected{
					headers: http.Header{
						headers.AccessControlAllowOrigin:      []string{"https://example.com"},
						headers.AccessControlAllowCredentials: defaultControlAllowCredentials,
						headers.AccessControlAllowHeaders:     defaultControlAllowHeaders,
						headers.AccessControlAllowMethods:     defaultControlAllowMethods,
						headers.AccessControlMaxAge:           defaultControlMaxAge,
						headers.AccessControlExposeHeaders:    defaultControlExposeHeaders,
						headers.AcceptLanguage:                []string{"en"},
					},
					code: http.StatusOK,
				},
			},
			{
				name: "custom code",
				args: args{
					code: http.StatusInternalServerError,
				},
				expected: expected{
					headers: defaultCorsHeaders,
					code:    http.StatusInternalServerError,
				},
			},
		}

		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				middleware := options.NewMiddleware(
					options.WithLogger(loggerMock),
					options.WithHeaders(testCase.args.headers),
					options.WithCode(testCase.args.code),
				)

				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodOptions, "/", nil)
				if testCase.args.requestHeaders != nil {
					request.Header = testCase.args.requestHeaders
				}

				middleware.Wrap(mockedNextHandler).
					ServeHTTP(
						contracts.WrapResponseWriter(recorder),
						request,
					)

				assert.Equal(t, testCase.expected.code, recorder.Code)
				assert.Equal(t, testCase.expected.headers, recorder.Header())
			})
		}
	})

	t.Run("for OPTIONS request with origin", func(t *testing.T) {
		mockedNextHandler := mocks.FailNowHandlerMock(t)

		testOrigin := "https://example.com"
		middleware := options.NewMiddleware(
			options.WithLogger(loggerMock),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodOptions, "/", nil)
		request.Header.Set(headers.Origin, testOrigin)

		middleware.Wrap(mockedNextHandler).
			ServeHTTP(
				contracts.WrapResponseWriter(recorder),
				request,
			)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, testOrigin, recorder.Header().Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "true", recorder.Header().Get(headers.AccessControlAllowCredentials))
		assert.Equal(t, "*", recorder.Header().Get(headers.AccessControlAllowHeaders))
		assert.Equal(t, "GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS", recorder.Header().Get(headers.AccessControlAllowMethods))
		assert.Equal(t, "86400", recorder.Header().Get(headers.AccessControlMaxAge))
		assert.Equal(t, "*", recorder.Header().Get(headers.AccessControlExposeHeaders))
	})

	t.Run("for OPTIONS request with empty origin", func(t *testing.T) {
		mockedNextHandler := mocks.FailNowHandlerMock(t)

		middleware := options.NewMiddleware(
			options.WithLogger(loggerMock),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodOptions, "/", nil)
		request.Header.Set(headers.Origin, "")

		middleware.Wrap(mockedNextHandler).
			ServeHTTP(
				contracts.WrapResponseWriter(recorder),
				request,
			)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "*", recorder.Header().Get(headers.AccessControlAllowOrigin))
	})

	t.Run("for OPTIONS request with origin and requested headers", func(t *testing.T) {
		mockedNextHandler := mocks.FailNowHandlerMock(t)

		testOrigin := "https://api.example.com"
		testHeaders := "X-Custom-Header, X-Another-Header"
		middleware := options.NewMiddleware(
			options.WithLogger(loggerMock),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodOptions, "/", nil)
		request.Header.Set(headers.Origin, testOrigin)
		request.Header.Set(headers.AccessControlRequestHeaders, testHeaders)

		middleware.Wrap(mockedNextHandler).
			ServeHTTP(
				contracts.WrapResponseWriter(recorder),
				request,
			)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, testOrigin, recorder.Header().Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, testHeaders, recorder.Header().Get(headers.AccessControlAllowHeaders))
	})

	t.Run("for OPTIONS request with all request headers", func(t *testing.T) {
		mockedNextHandler := mocks.FailNowHandlerMock(t)

		testOrigin := "https://test.example.com"
		testHeaders := "Content-Type, Authorization, X-Api-Key"
		testMethod := "DELETE"
		middleware := options.NewMiddleware(
			options.WithLogger(loggerMock),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodOptions, "/", nil)
		request.Header.Set(headers.Origin, testOrigin)
		request.Header.Set(headers.AccessControlRequestHeaders, testHeaders)
		request.Header.Set(headers.AccessControlRequestMethod, testMethod)

		middleware.Wrap(mockedNextHandler).
			ServeHTTP(
				contracts.WrapResponseWriter(recorder),
				request,
			)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, testOrigin, recorder.Header().Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, testHeaders, recorder.Header().Get(headers.AccessControlAllowHeaders))
		assert.Equal(t, testMethod, recorder.Header().Get(headers.AccessControlAllowMethods))
		assert.Equal(t, "86400", recorder.Header().Get(headers.AccessControlMaxAge))
		assert.Equal(t, "*", recorder.Header().Get(headers.AccessControlExposeHeaders))
	})

	t.Run("for non-OPTIONS request should call next", func(t *testing.T) {
		cases := []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodPatch,
			http.MethodHead,
			http.MethodTrace,
			http.MethodConnect,
		}

		for _, method := range cases {
			t.Run(method, func(t *testing.T) {
				mockedNextHandler := mocks.NewHandlerMock(t)

				middleware := options.NewMiddleware(
					options.WithLogger(loggerMock),
				)

				recorder := httptest.NewRecorder()
				response := contracts.WrapResponseWriter(recorder)
				request := httptest.NewRequest(method, "/", nil)

				mockedNextHandler.ServeHTTPMock.Expect(response, request)

				middleware.Wrap(mockedNextHandler).
					ServeHTTP(response, request)

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Empty(t, recorder.Header())
			})
		}
	})
}
