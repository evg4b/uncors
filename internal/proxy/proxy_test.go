package proxy_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/processor"
	"github.com/evg4b/uncors/internal/proxy"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestProxyMiddlewareWrap(t *testing.T) {
	replacerFactory, err := urlreplacer.NewURLReplacerFactory(map[string]string{
		"http://premium.local.com": "https://premium.api.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("should correctly replce headers in request to target resource", func(t *testing.T) {
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
				targetURL, err := url.Parse(testCase.URL)
				if err != nil {
					t.Fatal(err)
				}

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

				proc := processor.NewRequestProcessor(
					processor.WithMiddleware(proxy.NewProxyMiddleware(
						proxy.WithHTTPClient(httpClient),
						proxy.WithURLReplacerFactory(replacerFactory),
					)),
				)

				req, err := http.NewRequestWithContext(context.TODO(), "POST", targetURL.Path, nil)
				if err != nil {
					t.Fatal(err)
				}

				req.URL.Scheme = targetURL.Scheme
				req.URL.Host = targetURL.Host
				req.URL.Path = targetURL.Path

				req.Header.Add(testCase.headerKey, testCase.URL)

				http.HandlerFunc(proc.HandleRequest).
					ServeHTTP(httptest.NewRecorder(), req)
			})
		}
	})

	t.Run("should correctly replce headers in response", func(t *testing.T) {
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
				expectedURL, err := url.Parse(testCase.expectedURL)
				if err != nil {
					t.Fatal(err)
				}

				httpClient := testutils.NewTestClient(func(req *http.Request) *http.Response {
					return &http.Response{
						Status:     "200 OK",
						StatusCode: http.StatusOK,
						Header: http.Header{
							testCase.headerKey: {testCase.URL},
						},
						Body:          io.NopCloser(strings.NewReader("")),
						ContentLength: 0,
						Request:       req,
					}
				})

				proc := processor.NewRequestProcessor(
					processor.WithMiddleware(proxy.NewProxyMiddleware(
						proxy.WithHTTPClient(httpClient),
						proxy.WithURLReplacerFactory(replacerFactory),
					)),
				)

				req, err := http.NewRequestWithContext(context.TODO(), "POST", expectedURL.Path, nil)
				if err != nil {
					t.Fatal(err)
				}

				req.URL.Scheme = expectedURL.Scheme
				req.URL.Host = expectedURL.Host
				req.URL.Path = expectedURL.Path

				recorder := httptest.NewRecorder()

				http.HandlerFunc(proc.HandleRequest).
					ServeHTTP(recorder, req)

				assert.Equal(t, testCase.expectedURL, recorder.Header().Get(testCase.headerKey))
			})
		}
	})

	t.Run("should write allow CORS headers", func(t *testing.T) {
		httpClient := testutils.NewTestClient(func(req *http.Request) *http.Response {
			return &http.Response{
				Status:        "200 OK",
				StatusCode:    http.StatusOK,
				Header:        http.Header{},
				Body:          io.NopCloser(strings.NewReader("")),
				ContentLength: 0,
				Request:       req,
			}
		})

		proc := processor.NewRequestProcessor(
			processor.WithMiddleware(proxy.NewProxyMiddleware(
				proxy.WithHTTPClient(httpClient),
				proxy.WithURLReplacerFactory(replacerFactory),
			)),
		)

		req, err := http.NewRequestWithContext(context.TODO(), "POST", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		req.URL.Scheme = "http"
		req.URL.Host = "premium.local.com"

		recorder := httptest.NewRecorder()

		http.HandlerFunc(proc.HandleRequest).
			ServeHTTP(recorder, req)

		headers := recorder.Header()

		assert.Equal(t, "*", headers.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", headers.Get("Access-Control-Allow-Credentials"))
		assert.Equal(
			t,
			"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
			headers.Get("Access-Control-Allow-Methods"),
		)
	})
}
