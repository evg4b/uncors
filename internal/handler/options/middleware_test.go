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

		defaultCorsHeaders := http.Header{
			headers.AccessControlAllowOrigin:      defaultControlAllowOrigin,
			headers.AccessControlAllowCredentials: defaultControlAllowCredentials,
			headers.AccessControlAllowMethods:     defaultControlAllowMethods,
		}

		type args struct {
			headers map[string]string
			code    int
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
						headers.AccessControlAllowMethods:     defaultControlAllowMethods,
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
						headers.AccessControlAllowMethods:     defaultControlAllowMethods,
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

				middleware.Wrap(mockedNextHandler).
					ServeHTTP(
						contracts.WrapResponseWriter(recorder),
						httptest.NewRequest(http.MethodOptions, "/", nil),
					)

				assert.Equal(t, testCase.expected.code, recorder.Code)
				assert.Equal(t, testCase.expected.headers, recorder.Header())
			})
		}
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
