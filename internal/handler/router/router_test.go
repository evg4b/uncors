package router_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/engine"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/handler/router"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	mock1Body = `{"mock": "mock number 1"}`
	mock2Body = `{"mock": "mock number 2"}`
	mock3Body = `{"mock": "mock number 3"}`
	mock4Body = `{"mock": "mock number 4"}`

	backgroundPng = "background.png"
	iconsSvg      = "icons.svg"
	indexJS       = "index.js"
	styleCSS      = "styles.css"
	indexHTML     = "index.html"
	mockJSON      = "mock.json"

	api     = "http://localhost/api"
	apiUser = "https://localhost/api/user"

	userPath = "/api/user"

	userIDHeader = "User-Id"
)

func newGeneration(t *testing.T, fs afero.Fs) *engine.Generation {
	t.Helper()

	generation := engine.New(engine.WithFs(fs)).BuildGeneration(&config.UncorsConfig{})

	t.Cleanup(func() {
		require.NoError(t, generation.Close())
	})

	return generation
}

func cacheFactory() router.CacheMiddlewareFactory {
	return func(globs config.CacheGlobs) contracts.Middleware {
		return cache.NewMiddleware(
			cache.WithGlobs(globs),
			cache.WithCacheStorage(cache.NewRistrettoCache(100, time.Minute)),
		)
	}
}

func proxyFactory(
	t *testing.T,
	replacerFactory urlreplacer.ReplacerFactory,
	httpClient contracts.HTTPClient,
) contracts.Handler {
	if replacerFactory == nil {
		replacerFactory = mocks.NewReplacerFactoryMock(t)
	}

	if httpClient == nil {
		httpClient = mocks.NewHTTPClientMock(t)
	}

	return proxy.NewProxyHandler(
		proxy.WithURLReplacerFactory(replacerFactory),
		proxy.WithHTTPClient(httpClient),
	)
}

func serveHTTP(_ *testing.T, router http.Handler, recorder *httptest.ResponseRecorder, request *http.Request) {
	router.ServeHTTP(server.NewResponseRecorder(recorder), request)
}

func serve(t *testing.T, router http.Handler, method, url string, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	if strings.HasPrefix(url, "/") {
		url = "http://localhost" + url
	}

	request := httptest.NewRequestWithContext(t.Context(), method, url, nil)
	for key, value := range header {
		request.Header.Add(key, value)
	}

	helpers.NormaliseRequest(request)

	recorder := httptest.NewRecorder()
	serveHTTP(t, router, recorder, request)

	return recorder
}

func newRouter(
	t *testing.T,
	fs afero.Fs,
	mappings config.Mappings,
	httpClient contracts.HTTPClient,
	options ...router.Option,
) http.Handler {
	t.Helper()

	var replacerFactory urlreplacer.ReplacerFactory
	if httpClient != nil {
		replacerFactory = urlreplacer.NewURLReplacerFactory(mappings)
	}

	options = append([]router.Option{
		router.ForRouterWithDefaultHandler(proxyFactory(t, replacerFactory, httpClient)),
		router.ForRouterWithCacheMiddlewareFactory(cacheFactory()),
		router.WithFactory(newGeneration(t, fs)),
	}, options...)

	return router.NewRouter(mappings, options...)
}

func upstream(t *testing.T, code int, body string) contracts.HTTPClient {
	t.Helper()

	return mocks.NewHTTPClientMock(t).DoMock.Set(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Request:    req,
			StatusCode: code,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})
}

func mockResponse(path string, code int, body string) config.Mock {
	return config.Mock{
		Matcher:  config.RequestMatcher{Path: path},
		Response: config.Response{Code: code, Raw: body},
	}
}

type routeCase struct {
	name       string
	method     string
	url        string
	headers    map[string]string
	statusCode int
	expected   string
}

func (c routeCase) run(t *testing.T, router http.Handler) {
	t.Helper()

	method := c.method
	if method == "" {
		method = http.MethodGet
	}

	recorder := serve(t, router, method, c.url, c.headers)

	assert.Equal(t, c.statusCode, recorder.Code)
	assert.Equal(t, c.expected, testutils.ReadBody(t, recorder))
}

func runRouteCases(t *testing.T, router http.Handler, cases []routeCase) {
	t.Helper()

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.run(t, router)
		})
	}
}

func TestRouter(t *testing.T) {
	fs := testutils.FsFromMap(t, map[string]string{
		"/images/background.png": backgroundPng,
		"/images/svg/icons.svg":  iconsSvg,
		"/assets/js/index.js":    indexJS,
		"/assets/css/styles.css": styleCSS,
		"/assets/index.html":     indexHTML,
		"/mock.json":             mockJSON,
	})

	mappings := config.Mappings{
		{
			From: hosts.Localhost.HTTP(),
			To:   hosts.Localhost.HTTPS(),
			Statics: []config.StaticDirectory{
				{Dir: "/assets", Path: "/cc/", Index: indexHTML},
				{Dir: "/assets", Path: "/pnp/", Index: "index.php"},
				{Dir: "/images", Path: "/img/"},
			},
			Mocks: config.Mocks{
				mockResponse("/api/mocks/1", http.StatusOK, "mock-1"),
				{
					Matcher:  config.RequestMatcher{Path: "/api/mocks/2"},
					Response: config.Response{Code: http.StatusOK, File: "/mock.json"},
				},
				mockResponse("/api/mocks/3", http.StatusMultiStatus, "mock-3"),
				{
					Matcher:  config.RequestMatcher{Path: "/api/mocks/4"},
					Response: config.Response{Code: http.StatusOK, File: "/unknown.json"},
				},
			},
		},
	}

	routerInstance := newRouter(t, fs, mappings, upstream(t, http.StatusOK, "original.png"))

	runRouteCases(t, routerInstance, []routeCase{
		{name: "static index.html", url: "/cc/index.html", statusCode: http.StatusOK, expected: indexHTML},
		{name: "static index.js", url: "/cc/js/index.js", statusCode: http.StatusOK, expected: indexJS},
		{name: "static styles.css", url: "/cc/css/styles.css", statusCode: http.StatusOK, expected: styleCSS},
		{name: "static index fallback", url: "/cc/unknown.html", statusCode: http.StatusOK, expected: indexHTML},
		{name: "static background.png", url: "/img/background.png", statusCode: http.StatusOK, expected: backgroundPng},
		{name: "static icons.svg", url: "/img/svg/icons.svg", statusCode: http.StatusOK, expected: iconsSvg},
		{name: "missing static is proxied", url: "/img/original.png", statusCode: http.StatusOK, expected: "original.png"},
		{name: "raw content mock", url: "/api/mocks/1", statusCode: http.StatusOK, expected: "mock-1"},
		{name: "file content mock", url: "/api/mocks/2", statusCode: http.StatusOK, expected: mockJSON},
		{name: "MultiStatus mock", url: "/api/mocks/3", statusCode: http.StatusMultiStatus, expected: "mock-3"},
	})

	errorCases := []struct {
		name    string
		url     string
		message string
	}{
		{
			name:    "missing index file",
			url:     "/pnp/unknown.html",
			message: "failed to open index file: open /assets/index.php: file does not exist",
		},
		{
			name:    "missing mock file",
			url:     "/api/mocks/4",
			message: "failed to open file /unknown.json: open /unknown.json: file does not exist",
		},
	}
	for _, testCase := range errorCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := serve(t, routerInstance, http.MethodGet, testCase.url, nil)

			assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			assert.Contains(t, testutils.ReadBody(t, recorder), testCase.message)
		})
	}
}

func TestRouterScripts(t *testing.T) {
	routerInstance := newRouter(t, afero.NewMemMapFs(), config.Mappings{{
		From: hosts.Parse("{host}"),
		To:   hosts.Parse("{host}"),
		Scripts: config.Scripts{{
			Matcher: config.RequestMatcher{Path: "/api/script"},
			Script:  "response:WriteHeader(200)\nresponse:WriteString(\"from-script\")",
		}},
	}}, nil)

	runRouteCases(t, routerInstance, []routeCase{
		{name: "script", url: "/api/script", statusCode: http.StatusOK, expected: "from-script"},
	})
}

func TestRouterRewrites(t *testing.T) {
	routerInstance := newRouter(t, afero.NewMemMapFs(), config.Mappings{{
		From:     hosts.Parse("{host}"),
		To:       hosts.Parse("{host}"),
		Rewrites: config.RewriteOptions{{From: "/old", To: "/new"}},
	}}, upstream(t, http.StatusAccepted, "rewritten"))

	runRouteCases(t, routerInstance, []routeCase{
		{name: "exact path", url: "/old", statusCode: http.StatusAccepted, expected: "rewritten"},
		{name: "sub path", url: "/old/sub", statusCode: http.StatusAccepted, expected: "rewritten"},
	})
}

func TestRouterCacheGlobs(t *testing.T) {
	callCount := 0
	routerInstance := newRouter(t, afero.NewMemMapFs(), config.Mappings{{
		From:  hosts.Parse("{host}"),
		To:    hosts.Parse("{host}"),
		Cache: config.CacheGlobs{"*.json"},
		Mocks: config.Mocks{mockResponse("/data.json", http.StatusOK, `{"cached": true}`)},
	}}, nil, router.ForRouterWithCacheMiddlewareFactory(func(globs config.CacheGlobs) contracts.Middleware {
		callCount++

		return cacheFactory()(globs)
	}))

	recorder := serve(t, routerInstance, http.MethodGet, "http://localhost/data.json", nil)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, 1, callCount, "cache middleware factory should be called once for the mapping")
}

func TestRouterHAR(t *testing.T) {
	routerInstance := newRouter(t, afero.NewMemMapFs(), config.Mappings{{
		From:  hosts.Parse("{host}"),
		To:    hosts.Parse("{host}"),
		HAR:   config.HARConfig{File: filepath.Join(t.TempDir(), "test.har")},
		Mocks: config.Mocks{mockResponse("/api", http.StatusOK, mock1Body)},
	}}, nil)

	runRouteCases(t, routerInstance, []routeCase{
		{name: "mock", url: api, statusCode: http.StatusOK, expected: mock1Body},
	})
}

func TestRouterMockMethods(t *testing.T) {
	const (
		forwardedCode = 299
		forwardedBody = "forwarded"
	)

	t.Run("any method matches when the mock sets none", func(t *testing.T) {
		routerInstance := newRouter(t, afero.NewMemMapFs(), config.Mappings{{
			From:  hosts.Parse("{host}"),
			To:    hosts.Parse("{host}"),
			Mocks: config.Mocks{mockResponse("/api", http.StatusOK, mock1Body)},
		}}, nil)

		for _, method := range []string{
			http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
			http.MethodPatch, http.MethodDelete, http.MethodTrace,
		} {
			t.Run(method, func(t *testing.T) {
				routeCase{method: method, url: api, statusCode: http.StatusOK, expected: mock1Body}.run(t, routerInstance)
			})
		}
	})

	t.Run("only the set method matches", func(t *testing.T) {
		mock := mockResponse("/api", http.StatusOK, mock1Body)
		mock.Matcher.Method = http.MethodPut

		routerInstance := newRouter(t, afero.NewMemMapFs(), config.Mappings{{
			From:  hosts.Parse("{host}"),
			To:    hosts.Parse("{host}"),
			Mocks: config.Mocks{mock},
		}}, upstream(t, forwardedCode, forwardedBody))

		for _, method := range []string{
			http.MethodGet, http.MethodHead, http.MethodPost,
			http.MethodPatch, http.MethodDelete, http.MethodTrace,
		} {
			t.Run(method, func(t *testing.T) {
				routeCase{method: method, url: api, statusCode: forwardedCode, expected: forwardedBody}.run(t, routerInstance)
			})
		}

		runRouteCases(t, routerInstance, []routeCase{
			{name: http.MethodOptions, method: http.MethodOptions, url: api, statusCode: http.StatusOK},
			{name: http.MethodPut, method: http.MethodPut, url: api, statusCode: http.StatusOK, expected: mock1Body},
		})
	})
}

func TestRouterMockPaths(t *testing.T) {
	const (
		forwardedCode = 299
		forwardedBody = "forwarded"
	)

	routerInstance := newRouter(t, afero.NewMemMapFs(), config.Mappings{{
		From: hosts.Parse("{host}"),
		To:   hosts.Parse("{host}"),
		Mocks: config.Mocks{
			mockResponse(userPath, http.StatusOK, mock1Body),
			mockResponse("/api/user/{id:[0-9]+}", http.StatusAccepted, mock2Body),
			mockResponse("/api/{single-path/demo", http.StatusBadRequest, mock3Body),
			mockResponse(`/api/v2/{multiple-path:[a-z-\/]+}/demo`, http.StatusCreated, mock4Body),
		},
	}}, upstream(t, forwardedCode, forwardedBody))

	runRouteCases(t, routerInstance, []routeCase{
		{name: "direct path", url: apiUser, statusCode: http.StatusOK, expected: mock1Body},
		{
			name:       "direct path with ending slash",
			url:        "https://localhost/api/user/",
			statusCode: forwardedCode,
			expected:   forwardedBody,
		},
		{
			name:       "direct path with parameter",
			url:        "https://localhost/api/user/23",
			statusCode: http.StatusAccepted,
			expected:   mock2Body,
		},
		{
			name:       "direct path with incorrect parameter",
			url:        "https://localhost/api/user/unknow",
			statusCode: forwardedCode,
			expected:   forwardedBody,
		},
		{
			name:       "path with subpath to single matching param",
			url:        "https://localhost/api/some-path/with-some-subpath/demo",
			statusCode: forwardedCode,
			expected:   forwardedBody,
		},
		{
			name:       "path with subpath to multiple matching param",
			url:        "https://localhost/api/v2/some-path/with-some-subpath/demo",
			statusCode: http.StatusCreated,
			expected:   mock4Body,
		},
	})
}

func TestRouterMockQueries(t *testing.T) {
	const token = "000000000000000000000000000000"

	byID := mockResponse(userPath, http.StatusCreated, mock2Body)
	byID.Matcher.Queries = map[string]string{"id": "17"}

	byIDAndToken := mockResponse(userPath, http.StatusAccepted, mock3Body)
	byIDAndToken.Matcher.Queries = map[string]string{"id": "99", "token": token}

	routerInstance := newRouter(t, afero.NewMemMapFs(), config.Mappings{{
		From:  hosts.Parse("{host}"),
		To:    hosts.Parse("{host}"),
		Mocks: config.Mocks{mockResponse(userPath, http.StatusOK, mock1Body), byID, byIDAndToken},
	}}, nil)

	runRouteCases(t, routerInstance, []routeCase{
		{name: "queries is not set", url: "/api/user", statusCode: http.StatusOK, expected: mock1Body},
		{name: "passed unsetted parameter", url: "/api/user?id=16", statusCode: http.StatusOK, expected: mock1Body},
		{name: "passed parameter", url: "/api/user?id=17", statusCode: http.StatusCreated, expected: mock2Body},
		{
			name:       "passed one of multiple parameters",
			url:        "/api/user?id=99",
			statusCode: http.StatusOK,
			expected:   mock1Body,
		},
		{
			name:       "passed all of multiple parameters",
			url:        "/api/user?id=99&token=" + token,
			statusCode: http.StatusAccepted,
			expected:   mock3Body,
		},
		{
			name:       "passed extra parameters",
			url:        "/api/user?id=99&token=" + token + "&demo=true",
			statusCode: http.StatusAccepted,
			expected:   mock3Body,
		},
	})
}

func TestRouterMockHeaders(t *testing.T) {
	const (
		csrfToken = "de4e27987d054577b0edc0e828851724"
		token     = "000000000000000000000000000000"
	)

	byCSRF := mockResponse(userPath, http.StatusCreated, mock2Body)
	byCSRF.Matcher.Headers = map[string]string{headers.XCSRFToken: csrfToken}

	byUserAndCSRF := mockResponse(userPath, http.StatusAccepted, mock3Body)
	byUserAndCSRF.Matcher.Headers = map[string]string{userIDHeader: "99", headers.XCSRFToken: token}

	routerInstance := newRouter(t, afero.NewMemMapFs(), config.Mappings{{
		From:  hosts.Parse("{host}"),
		To:    hosts.Parse("{host}"),
		Mocks: config.Mocks{mockResponse(userPath, http.StatusOK, mock1Body), byCSRF, byUserAndCSRF},
	}}, nil)

	runRouteCases(t, routerInstance, []routeCase{
		{name: "headers is not set", method: http.MethodPost, url: apiUser, statusCode: http.StatusOK, expected: mock1Body},
		{
			name:       "passed unsetted headers",
			method:     http.MethodPost,
			url:        apiUser,
			headers:    map[string]string{headers.XCSRFToken: "55cc413b96026e833835a2c9a3f39c21"},
			statusCode: http.StatusOK,
			expected:   mock1Body,
		},
		{
			name:       "passed defined header",
			method:     http.MethodPost,
			url:        apiUser,
			headers:    map[string]string{headers.XCSRFToken: csrfToken},
			statusCode: http.StatusCreated,
			expected:   mock2Body,
		},
		{
			name:       "passed one of multiple headers",
			method:     http.MethodPost,
			url:        apiUser,
			headers:    map[string]string{userIDHeader: "99"},
			statusCode: http.StatusOK,
			expected:   mock1Body,
		},
		{
			name:       "passed all of multiple headers",
			method:     http.MethodPost,
			url:        apiUser,
			headers:    map[string]string{userIDHeader: "99", headers.XCSRFToken: token},
			statusCode: http.StatusAccepted,
			expected:   mock3Body,
		},
		{
			name:       "passed extra headers",
			method:     http.MethodPost,
			url:        apiUser,
			headers:    map[string]string{userIDHeader: "99", headers.XCSRFToken: token, headers.AcceptEncoding: "deflate"},
			statusCode: http.StatusAccepted,
			expected:   mock3Body,
		},
	})
}
