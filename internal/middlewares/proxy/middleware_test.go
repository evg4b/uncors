package proxy_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/middlewares/proxy"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

func TestProxyMiddleware(t *testing.T) {
	replacerFactory, err := urlreplacer.NewURLReplacerFactory(map[string]string{
		"http://premium.local.com": "https://premium.api.com",
	})
	testutils.CheckNoError(t, err)

	t.Run("should correctly replace headers in request to target resource", func(t *testing.T) {
		tests := []struct {
			name        string
			URL         string
			expectedURL string
			headerKey   string
		}{
			{
				name:        "transform Origin",
				URL:         "http://premium.local.com/app",
				expectedURL: "https://premium.api.com/app",
				headerKey:   headers.Origin,
			},
			{
				name:        "transform Referer",
				URL:         "http://premium.local.com/info",
				expectedURL: "https://premium.api.com/info",
				headerKey:   headers.Referer,
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				targetURL, err := urlx.Parse(testCase.URL)
				testutils.CheckNoError(t, err)

				httpClient := testutils.NewTestClient(func(req *http.Request) *http.Response {
					assert.Equal(t, testCase.expectedURL, req.Header.Get(testCase.headerKey))

					return &http.Response{
						Status:        "200 OK",
						StatusCode:    http.StatusOK,
						Header:        http.Header{},
						Body:          io.NopCloser(strings.NewReader("")),
						ContentLength: 0,
						Request:       req,
					}
				})

				proc := proxy.NewProxyMiddleware(
					proxy.WithHTTPClient(httpClient),
					proxy.WithURLReplacerFactory(replacerFactory),
					proxy.WithLogger(mocks.NewNoopLogger(t)),
				)

				req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, targetURL.Path, nil)
				testutils.CheckNoError(t, err)

				req.URL.Scheme = targetURL.Scheme
				req.Host = targetURL.Host
				req.URL.Path = targetURL.Path

				req.Header.Add(testCase.headerKey, testCase.URL)

				proc.ServeHTTP(httptest.NewRecorder(), req)
			})
		}
	})

	t.Run("should correctly replace headers in response", func(t *testing.T) {
		tests := []struct {
			name        string
			URL         string
			expectedURL string
			headerKey   string
		}{
			{
				name:        "transform Location",
				URL:         "https://premium.api.com/app",
				expectedURL: "http://premium.local.com/app",
				headerKey:   headers.Location,
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				expectedURL, err := urlx.Parse(testCase.expectedURL)
				testutils.CheckNoError(t, err)

				httpClient := testutils.NewTestClient(func(req *http.Request) *http.Response {
					return &http.Response{
						Status:     http.StatusText(http.StatusOK),
						StatusCode: http.StatusOK,
						Header: http.Header{
							testCase.headerKey: {testCase.URL},
						},
						Body:          io.NopCloser(strings.NewReader("")),
						ContentLength: 0,
						Request:       req,
					}
				})

				proc := proxy.NewProxyMiddleware(
					proxy.WithHTTPClient(httpClient),
					proxy.WithURLReplacerFactory(replacerFactory),
					proxy.WithLogger(mocks.NewNoopLogger(t)),
				)

				req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, expectedURL.Path, nil)
				testutils.CheckNoError(t, err)

				req.URL.Scheme = expectedURL.Scheme
				req.Host = expectedURL.Host
				req.URL.Path = expectedURL.Path

				recorder := httptest.NewRecorder()

				proc.ServeHTTP(recorder, req)

				assert.Equal(t, testCase.expectedURL, recorder.Header().Get(testCase.headerKey))
			})
		}
	})

	t.Run("should write allow CORS headers", func(t *testing.T) {
		httpClient := testutils.NewTestClient(func(req *http.Request) *http.Response {
			return &http.Response{
				Status:        http.StatusText(http.StatusOK),
				StatusCode:    http.StatusOK,
				Header:        http.Header{},
				Body:          io.NopCloser(strings.NewReader("")),
				ContentLength: 0,
				Request:       req,
			}
		})

		proc := proxy.NewProxyMiddleware(
			proxy.WithHTTPClient(httpClient),
			proxy.WithURLReplacerFactory(replacerFactory),
			proxy.WithLogger(mocks.NewNoopLogger(t)),
		)

		req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/", nil)
		testutils.CheckNoError(t, err)

		req.URL.Scheme = "http"
		req.Host = "premium.local.com"

		recorder := httptest.NewRecorder()

		proc.ServeHTTP(recorder, req)

		header := recorder.Header()
		assert.Equal(t, "*", header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "true", header.Get(headers.AccessControlAllowCredentials))
		assert.Equal(
			t,
			mocks.AllMethods,
			header.Get(headers.AccessControlAllowMethods),
		)
	})

	t.Run("OPTIONS request handling", func(t *testing.T) {
		middleware := proxy.NewProxyMiddleware(
			proxy.WithHTTPClient(http.DefaultClient),
			proxy.WithURLReplacerFactory(replacerFactory),
			proxy.WithLogger(mocks.NewNoopLogger(t)),
		)

		t.Run("should correctly create response", func(t *testing.T) {
			tests := []struct {
				name            string
				recorderFactory func() *httptest.ResponseRecorder
				expected        http.Header
			}{
				{
					name:            "should append data in empty writer",
					recorderFactory: httptest.NewRecorder,
					expected: map[string][]string{
						headers.AccessControlAllowOrigin:      {"*"},
						headers.AccessControlAllowCredentials: {"true"},
						headers.AccessControlAllowMethods:     {mocks.AllMethods},
					},
				},
				{
					name: "should append data in filled writer",
					recorderFactory: func() *httptest.ResponseRecorder {
						writer := httptest.NewRecorder()
						writer.Header().Set("Test-Header", "true")
						writer.Header().Set("X-Hey-Header", "123")

						return writer
					},
					expected: map[string][]string{
						"Test-Header":                         {"true"},
						"X-Hey-Header":                        {"123"},
						headers.AccessControlAllowOrigin:      {"*"},
						headers.AccessControlAllowCredentials: {"true"},
						headers.AccessControlAllowMethods:     {mocks.AllMethods},
					},
				},
				{
					name: "should override same headers",
					recorderFactory: func() *httptest.ResponseRecorder {
						writer := httptest.NewRecorder()
						writer.Header().Set("Custom-Header", "true")
						writer.Header().Set(headers.AccessControlAllowOrigin, "localhost:3000")

						return writer
					},
					expected: map[string][]string{
						"Custom-Header":                       {"true"},
						headers.AccessControlAllowOrigin:      {"*"},
						headers.AccessControlAllowCredentials: {"true"},
						headers.AccessControlAllowMethods:     {mocks.AllMethods},
					},
				},
			}
			for _, testCase := range tests {
				t.Run(testCase.name, func(t *testing.T) {
					recorder := testCase.recorderFactory()
					req, err := http.NewRequestWithContext(context.TODO(), http.MethodOptions, "/", nil)
					testutils.CheckNoError(t, err)

					middleware.ServeHTTP(recorder, req)

					assert.Equal(t, http.StatusOK, recorder.Code)
					assert.Equal(t, testCase.expected, recorder.Header())
				})
			}
		})
	})
}
