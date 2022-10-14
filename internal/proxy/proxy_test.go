package proxy_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/proxy"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestProxyHandler(t *testing.T) {
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
				headerKey:   "Origin",
			},
			{
				name:        "transform Referer",
				URL:         "http://premium.local.com/info",
				expectedURL: "https://premium.api.com/info",
				headerKey:   "Referer",
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

				proc := proxy.NewProxyHandler(
					proxy.WithHTTPClient(httpClient),
					proxy.WithURLReplacerFactory(replacerFactory),
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
				headerKey:   "Location",
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

				proc := proxy.NewProxyHandler(
					proxy.WithHTTPClient(httpClient),
					proxy.WithURLReplacerFactory(replacerFactory),
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

		proc := proxy.NewProxyHandler(
			proxy.WithHTTPClient(httpClient),
			proxy.WithURLReplacerFactory(replacerFactory),
		)

		req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/", nil)
		testutils.CheckNoError(t, err)

		req.URL.Scheme = "http"
		req.Host = "premium.local.com"

		recorder := httptest.NewRecorder()

		proc.ServeHTTP(recorder, req)

		headers := recorder.Header()

		assert.Equal(t, "*", headers.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", headers.Get("Access-Control-Allow-Credentials"))
		assert.Equal(
			t,
			"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
			headers.Get("Access-Control-Allow-Methods"),
		)
	})

	t.Run("OPTIONS request handling", func(t *testing.T) {
		handler := proxy.NewProxyHandler()

		t.Run("should correctly create response", func(t *testing.T) {
			testMethods := []struct {
				name     string
				headers  http.Header
				expected http.Header
			}{
				{
					name:     "should do not change empty headers",
					headers:  http.Header(map[string][]string{}),
					expected: http.Header(map[string][]string{}),
				},
				{
					name: "should do not skip not access-control-request-* headers",
					headers: http.Header{
						"Host":          {"www.host.com"},
						"Content-Type":  {"application/json"},
						"Authorization": {"Bearer Token"},
					},
					expected: http.Header{},
				},
				{
					name: "should allow all access-control-request-* headers",
					headers: http.Header{
						"Access-Control-Request-Headers": {"X-PINGOTHER, Content-Type"},
						"Access-Control-Request-Method":  {http.MethodPost, http.MethodDelete},
					},
					expected: http.Header{
						"Access-Control-Allow-Headers": {"X-PINGOTHER, Content-Type"},
						"Access-Control-Allow-Method":  {http.MethodPost, http.MethodDelete},
					},
				},
			}
			for _, testCase := range testMethods {
				t.Run(testCase.name, func(t *testing.T) {
					req, err := http.NewRequestWithContext(context.TODO(), http.MethodOptions, "/", nil)
					testutils.CheckNoError(t, err)

					req.Header = testCase.headers

					recorder := httptest.NewRecorder()
					handler.ServeHTTP(recorder, req)

					assert.Equal(t, http.StatusOK, recorder.Code)
					assert.Equal(t, testCase.expected, recorder.Header())
				})
			}
		})
	})
}
