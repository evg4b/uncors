package proxy_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testconstants"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	premiumLocalHost   = "premium.local.com"
	premiumLocalScheme = "http"
)

var errNetworkError = errors.New("network error")

func TestProxyHandler(t *testing.T) {
	replacerFactory := urlreplacer.NewURLReplacerFactory(config.Mappings{
		{From: hosts.Parse("http://premium.local.com"), To: hosts.Parse("https://premium.api.com")},
	})

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
				targetURL, err := url.Parse(testCase.URL)
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

				handler := proxy.NewProxyHandler(
					proxy.WithHTTPClient(httpClient),
					proxy.WithURLReplacerFactory(replacerFactory),
					proxy.WithOutput(mocks.NoopOutput()),
				)

				req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, targetURL.Path, nil)
				testutils.CheckNoError(t, err)

				req.Host = targetURL.Host
				req.URL = targetURL

				req.Header.Add(testCase.headerKey, testCase.URL)

				handler.ServeHTTP(infra.NewResponseRecorder(httptest.NewRecorder()), req)
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
				name:        "transform location",
				URL:         "https://premium.api.com/app",
				expectedURL: "http://premium.local.com/app",
				headerKey:   headers.Location,
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				expectedURL, err := url.Parse(testCase.expectedURL)
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

				handler := proxy.NewProxyHandler(
					proxy.WithHTTPClient(httpClient),
					proxy.WithURLReplacerFactory(replacerFactory),
					proxy.WithOutput(mocks.NoopOutput()),
				)

				req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, expectedURL.Path, nil)
				testutils.CheckNoError(t, err)

				req.URL.Scheme = expectedURL.Scheme
				req.Host = expectedURL.Host
				req.URL.Path = expectedURL.Path
				infra.NormaliseRequest(req)

				recorder := httptest.NewRecorder()

				handler.ServeHTTP(infra.NewResponseRecorder(recorder), req)

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

		handler := proxy.NewProxyHandler(
			proxy.WithHTTPClient(httpClient),
			proxy.WithURLReplacerFactory(replacerFactory),
			proxy.WithOutput(mocks.NoopOutput()),
		)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)
		testutils.CheckNoError(t, err)

		req.URL.Scheme = premiumLocalScheme
		req.Host = premiumLocalHost
		infra.NormaliseRequest(req)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(infra.NewResponseRecorder(recorder), req)

		header := recorder.Header()
		assert.Equal(t, "*", header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "true", header.Get(headers.AccessControlAllowCredentials))
		assert.Equal(
			t,
			testconstants.AllMethods,
			header.Get(headers.AccessControlAllowMethods),
		)
	})

	t.Run("should forward non-mapped headers unchanged", func(t *testing.T) {
		httpClient := testutils.NewTestClient(func(req *http.Request) *http.Response {
			assert.Equal(t, "application/json", req.Header.Get(headers.ContentType))

			return &http.Response{
				Status:        "200 OK",
				StatusCode:    http.StatusOK,
				Header:        http.Header{},
				Body:          io.NopCloser(strings.NewReader("")),
				ContentLength: 0,
				Request:       req,
			}
		})

		handler := proxy.NewProxyHandler(
			proxy.WithHTTPClient(httpClient),
			proxy.WithURLReplacerFactory(replacerFactory),
			proxy.WithOutput(mocks.NoopOutput()),
		)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://premium.local.com/", nil)
		require.NoError(t, err)

		req.URL.Scheme = premiumLocalScheme
		req.Host = premiumLocalHost
		req.Header.Set(headers.ContentType, "application/json")
		infra.NormaliseRequest(req)

		handler.ServeHTTP(infra.NewResponseRecorder(httptest.NewRecorder()), req)
	})

	t.Run("should forward cookies from request to target", func(t *testing.T) {
		httpClient := testutils.NewTestClient(func(req *http.Request) *http.Response {
			cookie, err := req.Cookie("session")
			require.NoError(t, err)
			assert.Equal(t, "abc123", cookie.Value)

			return &http.Response{
				Status:        "200 OK",
				StatusCode:    http.StatusOK,
				Header:        http.Header{},
				Body:          io.NopCloser(strings.NewReader("")),
				ContentLength: 0,
				Request:       req,
			}
		})

		handler := proxy.NewProxyHandler(
			proxy.WithHTTPClient(httpClient),
			proxy.WithURLReplacerFactory(replacerFactory),
			proxy.WithOutput(mocks.NoopOutput()),
		)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://premium.local.com/", nil)
		require.NoError(t, err)

		req.URL.Scheme = premiumLocalScheme
		req.Host = premiumLocalHost
		req.AddCookie(&http.Cookie{
			Name:     "session",
			Value:    "abc123",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
		infra.NormaliseRequest(req)

		handler.ServeHTTP(infra.NewResponseRecorder(httptest.NewRecorder()), req)
	})

	t.Run("should forward cookies from response to source", func(t *testing.T) {
		httpClient := testutils.NewTestClient(func(req *http.Request) *http.Response {
			return &http.Response{
				Status:     "200 OK",
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Set-Cookie": {"session=abc123; Path=/"},
				},
				Body:          io.NopCloser(strings.NewReader("")),
				ContentLength: 0,
				Request:       req,
			}
		})

		handler := proxy.NewProxyHandler(
			proxy.WithHTTPClient(httpClient),
			proxy.WithURLReplacerFactory(replacerFactory),
			proxy.WithOutput(mocks.NoopOutput()),
		)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://premium.local.com/", nil)
		require.NoError(t, err)

		req.URL.Scheme = premiumLocalScheme
		req.Host = premiumLocalHost
		infra.NormaliseRequest(req)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(infra.NewResponseRecorder(recorder), req)

		cookies := recorder.Result().Cookies()
		require.NotEmpty(t, cookies)
		assert.Equal(t, "abc123", cookies[0].Value)
	})

	t.Run("should proxy request using rewrite host from context", func(t *testing.T) {
		httpClient := testutils.NewTestClient(func(req *http.Request) *http.Response {
			assert.Equal(t, "premium.api.com", req.URL.Host)

			return &http.Response{
				Status:        "200 OK",
				StatusCode:    http.StatusOK,
				Header:        http.Header{},
				Body:          io.NopCloser(strings.NewReader("")),
				ContentLength: 0,
				Request:       req,
			}
		})

		handler := proxy.NewProxyHandler(
			proxy.WithHTTPClient(httpClient),
			proxy.WithURLReplacerFactory(replacerFactory),
			proxy.WithOutput(mocks.NoopOutput()),
		)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://premium.local.com/app", nil)
		require.NoError(t, err)

		req.URL.Scheme = premiumLocalScheme
		req.Host = premiumLocalHost
		infra.NormaliseRequest(req)
		req = req.WithContext(context.WithValue(req.Context(), rewrite.RewriteHostKey, "premium.api.com"))

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(infra.NewResponseRecorder(recorder), req)
	})

	t.Run("should answer 502 when the upstream request fails", func(t *testing.T) {
		httpMock := mocks.NewHTTPClientMock(t).DoMock.Set(func(_ *http.Request) (*http.Response, error) {
			return nil, errNetworkError
		})

		handler := proxy.NewProxyHandler(
			proxy.WithHTTPClient(httpMock),
			proxy.WithURLReplacerFactory(replacerFactory),
			proxy.WithOutput(mocks.NoopOutput()),
		)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://premium.local.com/", nil)
		require.NoError(t, err)

		req.URL.Scheme = premiumLocalScheme
		req.Host = premiumLocalHost
		infra.NormaliseRequest(req)

		recorder := httptest.NewRecorder()
		responseWriter := infra.NewResponseRecorder(recorder)

		require.NoError(t, handler.Serve(responseWriter, req))

		assert.Equal(t, http.StatusBadGateway, recorder.Code)
	})

	t.Run("OPTIONS request handling", func(t *testing.T) {
		t.Skip()

		handler := proxy.NewProxyHandler(
			proxy.WithHTTPClient(http.DefaultClient),
			proxy.WithURLReplacerFactory(replacerFactory),
			proxy.WithOutput(mocks.NoopOutput()),
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
						headers.AccessControlAllowMethods:     {testconstants.AllMethods},
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
						headers.AccessControlAllowMethods:     {testconstants.AllMethods},
					},
				},
				{
					name: "should override same headers",
					recorderFactory: func() *httptest.ResponseRecorder {
						writer := httptest.NewRecorder()
						writer.Header().Set("Custom-Header", "true")
						writer.Header().Set(headers.AccessControlAllowOrigin, hosts.Localhost.Port(3000).String())

						return writer
					},
					expected: map[string][]string{
						"Custom-Header":                       {"true"},
						headers.AccessControlAllowOrigin:      {"*"},
						headers.AccessControlAllowCredentials: {"true"},
						headers.AccessControlAllowMethods:     {testconstants.AllMethods},
					},
				},
			}
			for _, testCase := range tests {
				t.Run(testCase.name, func(t *testing.T) {
					recorder := testCase.recorderFactory()
					req, err := http.NewRequestWithContext(t.Context(), http.MethodOptions, "/", nil)
					testutils.CheckNoError(t, err)

					handler.ServeHTTP(infra.NewResponseRecorder(recorder), req)

					assert.Equal(t, http.StatusOK, recorder.Code)
					assert.Equal(t, testCase.expected, recorder.Header())
				})
			}
		})
	})
}

func TestProxyHandlerForwarding(t *testing.T) {
	replacerFactory := urlreplacer.NewURLReplacerFactory(config.Mappings{
		{From: hosts.Parse("http://premium.local.com"), To: hosts.Parse("https://premium.api.com")},
	})

	newRequest := func(t *testing.T) *http.Request {
		t.Helper()

		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://premium.local.com/app", nil)
		require.NoError(t, err)

		request.Host = "premium.local.com"
		request.RemoteAddr = "10.1.2.3:4567"
		infra.NormaliseRequest(request)

		return request
	}

	serve := func(t *testing.T, request *http.Request, inspect func(*http.Request)) *httptest.ResponseRecorder {
		t.Helper()

		handler := proxy.NewProxyHandler(
			proxy.WithHTTPClient(testutils.NewTestClient(func(outbound *http.Request) *http.Response {
				inspect(outbound)

				return &http.Response{
					Status:     http.StatusText(http.StatusOK),
					StatusCode: http.StatusOK,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader("")),
					Request:    outbound,
				}
			})),
			proxy.WithURLReplacerFactory(replacerFactory),
			proxy.WithOutput(mocks.NoopOutput()),
		)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(infra.NewResponseRecorder(recorder), request)

		return recorder
	}

	t.Run("does not forward hop-by-hop headers", func(t *testing.T) {
		request := newRequest(t)
		request.Header.Set("Connection", "keep-alive, X-Custom-Hop")
		request.Header.Set("X-Custom-Hop", "dropped")
		request.Header.Set(headers.ProxyAuthorization, "Basic dGVzdA==")

		serve(t, request, func(outbound *http.Request) {
			assert.Empty(t, outbound.Header.Get("Connection"))
			assert.Empty(t, outbound.Header.Get("X-Custom-Hop"))
			assert.Empty(t, outbound.Header.Get(headers.ProxyAuthorization))
		})
	})

	t.Run("announces the original client", func(t *testing.T) {
		serve(t, newRequest(t), func(outbound *http.Request) {
			assert.Equal(t, "10.1.2.3", outbound.Header.Get(headers.XForwardedFor))
			assert.Equal(t, "premium.local.com", outbound.Header.Get("X-Forwarded-Host"))
			assert.Equal(t, "http", outbound.Header.Get(headers.XForwardedProto))
		})
	})

	t.Run("keeps path and query while replacing the host", func(t *testing.T) {
		request := newRequest(t)
		request.URL.RawQuery = "q=a%20b&x=1"

		serve(t, request, func(outbound *http.Request) {
			assert.Equal(t, "premium.api.com", outbound.URL.Host)
			assert.Equal(t, "https", outbound.URL.Scheme)
			assert.Equal(t, "/app", outbound.URL.Path)
			assert.Equal(t, "q=a%20b&x=1", outbound.URL.RawQuery)
			assert.Equal(t, "premium.api.com", outbound.Host)
		})
	})
}
