package proxy_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testconstants"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
)

const (
	premiumLocalHost   = "premium.local.com"
	premiumLocalScheme = "http"
)

var errNetworkError = errors.New("network error")

func TestProxyHandler(t *testing.T) {
	replacerFactory := lo.Must(urlreplacer.NewURLReplacerFactory(config.Mappings{
		{From: hosts.Parse("http://premium.local.com"), To: hosts.Parse("https://premium.api.com")},
	}))

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
				targetURL, err := urlt.Parse(testCase.URL)
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

				handler := proxy.NewHandler(
					proxy.WithHTTPClient(httpClient),
					proxy.WithURLReplacerFactory(replacerFactory),
				)

				req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, targetURL.Path, http.NoBody)
				testutils.CheckNoError(t, err)

				req.Host = targetURL.Host
				req.URL = targetURL

				req.Header.Add(testCase.headerKey, testCase.URL)

				err = handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), req)
				require.NoError(t, err)
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
				expectedURL, err := urlt.Parse(testCase.expectedURL)
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

				handler := proxy.NewHandler(
					proxy.WithHTTPClient(httpClient),
					proxy.WithURLReplacerFactory(replacerFactory),
				)

				req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, expectedURL.Path, http.NoBody)
				testutils.CheckNoError(t, err)

				req.URL.Scheme = expectedURL.Scheme
				req.Host = expectedURL.Host
				req.URL.Path = expectedURL.Path
				helpers.NormaliseRequest(req)

				recorder := httptest.NewRecorder()

				err = handler.ServeHTTP(server.NewResponseRecorder(recorder), req)
				require.NoError(t, err)

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

		handler := proxy.NewHandler(
			proxy.WithHTTPClient(httpClient),
			proxy.WithURLReplacerFactory(replacerFactory),
		)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/", http.NoBody)
		testutils.CheckNoError(t, err)

		req.URL.Scheme = premiumLocalScheme
		req.Host = premiumLocalHost
		helpers.NormaliseRequest(req)

		recorder := httptest.NewRecorder()

		err = handler.ServeHTTP(server.NewResponseRecorder(recorder), req)
		require.NoError(t, err)

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

		handler := proxy.NewHandler(
			proxy.WithHTTPClient(httpClient),
			proxy.WithURLReplacerFactory(replacerFactory),
		)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://premium.local.com/", http.NoBody)
		require.NoError(t, err)

		req.URL.Scheme = premiumLocalScheme
		req.Host = premiumLocalHost
		req.Header.Set(headers.ContentType, "application/json")
		helpers.NormaliseRequest(req)

		err = handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), req)
		require.NoError(t, err)
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

		handler := proxy.NewHandler(
			proxy.WithHTTPClient(httpClient),
			proxy.WithURLReplacerFactory(replacerFactory),
		)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://premium.local.com/", http.NoBody)
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
		helpers.NormaliseRequest(req)

		err = handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), req)
		require.NoError(t, err)
	})

	t.Run("should forward Origin and Referer of other hosts unchanged", func(t *testing.T) {
		tests := []struct {
			name      string
			headerKey string
			value     string
		}{
			{name: "Origin of another host", headerKey: headers.Origin, value: "http://myapp.test:8080"},
			{name: "Origin of the loopback address", headerKey: headers.Origin, value: "http://127.0.0.1:3000"},
			{name: "null Origin", headerKey: headers.Origin, value: "null"},
			{name: "Referer of another host", headerKey: headers.Referer, value: "http://myapp.test/page?q=1"},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				var forwarded *http.Request

				handler := proxy.NewHandler(
					proxy.WithHTTPClient(capturingClient(&forwarded)),
					proxy.WithURLReplacerFactory(replacerFactory),
				)

				req := premiumRequest(t, http.MethodGet, nil)
				req.Header.Set(testCase.headerKey, testCase.value)

				recorder := httptest.NewRecorder()
				err := handler.ServeHTTP(server.NewResponseRecorder(recorder), req)
				require.NoError(t, err)

				require.NotNil(t, forwarded)
				assert.Equal(t, testCase.value, forwarded.Header.Get(testCase.headerKey))
				assert.Equal(t, http.StatusOK, recorder.Code)
			})
		}
	})

	t.Run("should forward the request Content-Length", func(t *testing.T) {
		var forwarded *http.Request

		handler := proxy.NewHandler(
			proxy.WithHTTPClient(capturingClient(&forwarded)),
			proxy.WithURLReplacerFactory(replacerFactory),
		)

		req := premiumRequest(t, http.MethodPost, io.NopCloser(strings.NewReader("hello")))
		req.ContentLength = 5

		err := handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), req)
		require.NoError(t, err)

		require.NotNil(t, forwarded)
		assert.Equal(t, int64(5), forwarded.ContentLength)
	})

	t.Run("should forward the Cookie header unchanged", func(t *testing.T) {
		const cookie = `quoted="a b"; list=x,y; spaced=with space; flag; session=abc123`

		var forwarded *http.Request

		handler := proxy.NewHandler(
			proxy.WithHTTPClient(capturingClient(&forwarded)),
			proxy.WithURLReplacerFactory(replacerFactory),
		)

		req := premiumRequest(t, http.MethodGet, nil)
		req.Header.Set(headers.Cookie, cookie)

		err := handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), req)
		require.NoError(t, err)

		require.NotNil(t, forwarded)
		assert.Equal(t, []string{cookie}, forwarded.Header.Values(headers.Cookie))
	})

	t.Run("should not forward hop-by-hop request headers", func(t *testing.T) {
		var forwarded *http.Request

		handler := proxy.NewHandler(
			proxy.WithHTTPClient(capturingClient(&forwarded)),
			proxy.WithURLReplacerFactory(replacerFactory),
		)

		req := premiumRequest(t, http.MethodGet, nil)
		req.Header.Set("Connection", "keep-alive, X-Hop")
		req.Header.Set("X-Hop", "1")
		req.Header.Set("Keep-Alive", "timeout=5")
		req.Header.Set("Proxy-Connection", "keep-alive")
		req.Header.Set("Proxy-Authorization", "Basic dXNlcjpwYXNz")
		req.Header.Set("TE", "trailers, deflate")
		req.Header.Set("Trailer", "X-Checksum")
		req.Header.Set("X-End-To-End", "kept")

		err := handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), req)
		require.NoError(t, err)

		require.NotNil(t, forwarded)

		for _, name := range []string{
			"Connection", "X-Hop", "Keep-Alive", "Proxy-Connection", "Proxy-Authorization", "Trailer", "Upgrade",
		} {
			assert.Empty(t, forwarded.Header.Values(name), name)
		}

		assert.Equal(t, "trailers", forwarded.Header.Get("TE"))
		assert.Equal(t, "kept", forwarded.Header.Get("X-End-To-End"))
	})

	t.Run("should keep the upgrade headers of an upgrade request", func(t *testing.T) {
		var forwarded *http.Request

		handler := proxy.NewHandler(
			proxy.WithHTTPClient(capturingClient(&forwarded)),
			proxy.WithURLReplacerFactory(replacerFactory),
		)

		req := premiumRequest(t, http.MethodGet, nil)
		req.Header.Set("Connection", "keep-alive, Upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Keep-Alive", "timeout=5")

		err := handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), req)
		require.NoError(t, err)

		require.NotNil(t, forwarded)
		assert.Equal(t, "Upgrade", forwarded.Header.Get("Connection"))
		assert.Equal(t, "websocket", forwarded.Header.Get("Upgrade"))
		assert.Empty(t, forwarded.Header.Values("Keep-Alive"))
	})

	t.Run("should not copy hop-by-hop response headers", func(t *testing.T) {
		handler := proxy.NewHandler(
			proxy.WithHTTPClient(testutils.NewTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Connection":   {"close, X-Hop"},
						"X-Hop":        {"1"},
						"Keep-Alive":   {"timeout=5"},
						"X-End-To-End": {"kept"},
					},
					Body:    io.NopCloser(strings.NewReader("body")),
					Request: req,
				}
			})),
			proxy.WithURLReplacerFactory(replacerFactory),
		)

		recorder := httptest.NewRecorder()
		err := handler.ServeHTTP(server.NewResponseRecorder(recorder), premiumRequest(t, http.MethodGet, nil))
		require.NoError(t, err)

		for _, name := range []string{"Connection", "X-Hop", "Keep-Alive"} {
			assert.Empty(t, recorder.Header().Values(name), name)
		}

		assert.Equal(t, "kept", recorder.Header().Get("X-End-To-End"))
	})

	t.Run("should keep Content-Length of responses without a body", func(t *testing.T) {
		tests := []struct {
			name   string
			method string
			status int
			want   string
		}{
			{name: "HEAD", method: http.MethodHead, status: http.StatusOK, want: "1234"},
			{name: "304", method: http.MethodGet, status: http.StatusNotModified, want: "1234"},
			{name: "GET with body", method: http.MethodGet, status: http.StatusOK, want: ""},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := proxy.NewHandler(
					proxy.WithHTTPClient(testutils.NewTestClient(func(req *http.Request) *http.Response {
						return &http.Response{
							StatusCode: testCase.status,
							Header:     http.Header{"Content-Length": {"1234"}},
							Body:       http.NoBody,
							Request:    req,
						}
					})),
					proxy.WithURLReplacerFactory(replacerFactory),
				)

				recorder := httptest.NewRecorder()
				err := handler.ServeHTTP(server.NewResponseRecorder(recorder), premiumRequest(t, testCase.method, nil))
				require.NoError(t, err)

				assert.Equal(t, testCase.want, recorder.Header().Get("Content-Length"))
			})
		}
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

		handler := proxy.NewHandler(
			proxy.WithHTTPClient(httpClient),
			proxy.WithURLReplacerFactory(replacerFactory),
		)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://premium.local.com/", http.NoBody)
		require.NoError(t, err)

		req.URL.Scheme = premiumLocalScheme
		req.Host = premiumLocalHost
		helpers.NormaliseRequest(req)

		recorder := httptest.NewRecorder()
		err = handler.ServeHTTP(server.NewResponseRecorder(recorder), req)
		require.NoError(t, err)

		cookies := recorder.Result().Cookies()
		require.NotEmpty(t, cookies)
		assert.Equal(t, "abc123", cookies[0].Value)
	})

	t.Run("should replace the cookie domain without the local port", func(t *testing.T) {
		tests := []struct {
			name   string
			from   string
			host   string
			domain string
			want   string
		}{
			{
				name:   "with a port",
				from:   "http://{tenant}.local.dev:3000",
				host:   "acme.local.dev:3000",
				domain: ".acme.api.com",
				want:   "Domain=acme.local.dev",
			},
			{
				name:   "without a port",
				from:   "http://{tenant}.local.dev",
				host:   "acme.local.dev",
				domain: "acme.api.com",
				want:   "Domain=acme.local.dev",
			},
			{
				name: "without a domain",
				from: "http://{tenant}.local.dev:3000",
				host: "acme.local.dev:3000",
				want: "a=1; Path=/",
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				setCookie := "a=1; Path=/"
				if testCase.domain != "" {
					setCookie += "; Domain=" + testCase.domain
				}

				handler := proxy.NewHandler(
					proxy.WithHTTPClient(testutils.NewTestClient(func(req *http.Request) *http.Response {
						return &http.Response{
							StatusCode: http.StatusOK,
							Header:     http.Header{headers.SetCookie: {setCookie}},
							Body:       http.NoBody,
							Request:    req,
						}
					})),
					proxy.WithURLReplacerFactory(lo.Must(urlreplacer.NewURLReplacerFactory(config.Mappings{
						{From: hosts.Parse(testCase.from), To: hosts.Parse("https://{tenant}.api.com")},
					}))),
				)

				req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+testCase.host+"/", http.NoBody)
				helpers.NormaliseRequest(req)

				recorder := httptest.NewRecorder()
				require.NoError(t, handler.ServeHTTP(server.NewResponseRecorder(recorder), req))

				assert.Contains(t, recorder.Header().Get(headers.SetCookie), testCase.want)
			})
		}
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

		handler := proxy.NewHandler(
			proxy.WithHTTPClient(httpClient),
			proxy.WithURLReplacerFactory(replacerFactory),
		)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://premium.local.com/app", http.NoBody)
		require.NoError(t, err)

		req.URL.Scheme = premiumLocalScheme
		req.Host = premiumLocalHost
		helpers.NormaliseRequest(req)
		req = req.WithContext(context.WithValue(req.Context(), rewrite.RewriteHostKey, "premium.api.com"))

		recorder := httptest.NewRecorder()
		err = handler.ServeHTTP(server.NewResponseRecorder(recorder), req)
		require.NoError(t, err)
	})

	t.Run("should return and trace error when http client fails", func(t *testing.T) {
		httpMock := mocks.NewHTTPClientMock(t).DoMock.Set(func(_ *http.Request) (*http.Response, error) {
			return nil, errNetworkError
		})

		spans := testutils.Telemetry(t).Spans

		handler := proxy.NewHandler(
			proxy.WithHTTPClient(httpMock),
			proxy.WithURLReplacerFactory(replacerFactory),
		)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://premium.local.com/", http.NoBody)
		require.NoError(t, err)

		req.URL.Scheme = premiumLocalScheme
		req.Host = premiumLocalHost
		helpers.NormaliseRequest(req)

		recorder := httptest.NewRecorder()
		responseWriter := server.NewResponseRecorder(recorder)

		handlerErr := handler.ServeHTTP(responseWriter, req)

		require.ErrorIs(t, handlerErr, errNetworkError)

		ended := spans.Ended()
		require.Len(t, ended, 1)
		assert.Equal(t, "proxy.forward", ended[0].Name())
		assert.Equal(t, codes.Error, ended[0].Status().Code)
		assert.Contains(t, ended[0].Status().Description, "network error")
	})

	t.Run("OPTIONS request handling", func(t *testing.T) {
		t.Skip()

		handler := proxy.NewHandler(
			proxy.WithHTTPClient(http.DefaultClient),
			proxy.WithURLReplacerFactory(replacerFactory),
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
					req, err := http.NewRequestWithContext(t.Context(), http.MethodOptions, "/", http.NoBody)
					testutils.CheckNoError(t, err)

					err = handler.ServeHTTP(server.NewResponseRecorder(recorder), req)
					require.NoError(t, err)

					assert.Equal(t, http.StatusOK, recorder.Code)
					assert.Equal(t, testCase.expected, recorder.Header())
				})
			}
		})
	})
}

func premiumRequest(t *testing.T, method string, body io.Reader) *http.Request {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), method, "http://premium.local.com/", body)
	require.NoError(t, err)

	req.URL.Scheme = premiumLocalScheme
	req.Host = premiumLocalHost
	helpers.NormaliseRequest(req)

	return req
}

func capturingClient(forwarded **http.Request) contracts.HTTPClient {
	return testutils.NewTestClient(func(req *http.Request) *http.Response {
		*forwarded = req

		return &http.Response{
			Status:     "200 OK",
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    req,
		}
	})
}
