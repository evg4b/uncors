package handler_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/evg4b/uncors/pkg/fakedata"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/internal/handler/options"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/handler/static"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	goCache "github.com/patrickmn/go-cache"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
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

func cacheFactory() handler.CacheMiddlewareFactory {
	return func(globs config.CacheGlobs) contracts.Middleware {
		return cache.NewMiddleware(
			cache.WithGlobs(globs),
			cache.WithLogger(log.New(io.Discard)),
			cache.WithCacheStorage(goCache.New(goCache.DefaultExpiration, goCache.DefaultExpiration)),
		)
	}
}

func proxyFactory(
	t *testing.T,
	replacerFactory urlreplacer.ReplacerFactory,
	httpClient contracts.HTTPClient,
) handler.ProxyHandlerFactory {
	if replacerFactory == nil {
		replacerFactory = mocks.NewReplacerFactoryMock(t)
	}

	if httpClient == nil {
		httpClient = mocks.NewHTTPClientMock(t)
	}

	return func() contracts.Handler {
		return proxy.NewProxyHandler(
			proxy.WithURLReplacerFactory(replacerFactory),
			proxy.WithHTTPClient(httpClient),
			proxy.WithProxyLogger(log.New(io.Discard)),
			proxy.WithRewriteLogger(log.New(io.Discard)),
		)
	}
}

func optionsFactory() handler.OptionsMiddlewareFactory {
	return func(config config.OptionsHandling) contracts.Middleware {
		return options.NewMiddleware(
			options.WithLogger(log.New(io.Discard)),
			options.WithHeaders(config.Headers),
			options.WithCode(config.Code),
		)
	}
}

func staticFactory(fs afero.Fs) handler.StaticMiddlewareFactory {
	return func(path string, dir config.StaticDirectory) contracts.Middleware {
		return static.NewStaticMiddleware(
			static.WithFileSystem(afero.NewBasePathFs(fs, dir.Dir)),
			static.WithIndex(dir.Index),
			static.WithLogger(log.New(io.Discard)),
			static.WithPrefix(path),
		)
	}
}

func mockFactory(fs afero.Fs) handler.MockHandlerFactory {
	if fs == nil {
		fs = afero.NewMemMapFs()
	}

	return func(response config.Response) contracts.Handler {
		return mock.NewMockHandler(
			mock.WithLogger(log.New(io.Discard)),
			mock.WithResponse(response),
			mock.WithFileSystem(fs),
			mock.WithAfter(time.After),
			mock.WithGenerator(fakedata.NewGoFakeItGenerator()),
		)
	}
}

func TestUncorsRequestHandler(t *testing.T) {
	log.SetOutput(io.Discard)
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
				{
					Path: "/api/mocks/1",
					Response: config.Response{
						Code: http.StatusOK,
						Raw:  "mock-1",
					},
				},
				{
					Path: "/api/mocks/2",
					Response: config.Response{
						Code: http.StatusOK,
						File: "/mock.json",
					},
				},
				{
					Path: "/api/mocks/3",
					Response: config.Response{
						Code: http.StatusMultiStatus,
						Raw:  "mock-3",
					},
				},
				{
					Path: "/api/mocks/4",
					Response: config.Response{
						Code: http.StatusOK,
						File: "/unknown.json",
					},
				},
			},
		},
	}

	factory := urlreplacer.NewURLReplacerFactory(mappings)

	httpResponseMapping := map[string]string{
		"/img/original.png": "original.png",
	}

	httpMock := mocks.NewHTTPClientMock(t).DoMock.Set(func(request *http.Request) (*http.Response, error) {
		if response, ok := httpResponseMapping[request.URL.Path]; ok {
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(response)),
				Status:     http.StatusText(http.StatusOK),
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Request:    request,
			}, nil
		}

		panic(helpers.Sprintf("incorrect request: %s", request.URL.Path))
	})

	uncorsHandler := handler.NewUncorsRequestHandler(
		handler.WithLogger(mocks.NewLoggerMock(t)),
		handler.WithMappings(mappings),
		handler.WithCacheMiddlewareFactory(cacheFactory()),
		handler.WithProxyHandlerFactory(proxyFactory(t, factory, httpMock)),
		handler.WithStaticHandlerFactory(staticFactory(fs)),
		handler.WithMockHandlerFactory(mockFactory(fs)),
		handler.WithOptionsHandlerFactory(optionsFactory()),
	)

	t.Run("statics directory", func(t *testing.T) {
		t.Run("with index file", func(t *testing.T) {
			t.Run("should return static file", func(t *testing.T) {
				tests := []struct {
					name     string
					url      string
					expected string
				}{
					{
						name:     indexHTML,
						url:      "http://localhost/cc/index.html",
						expected: indexHTML,
					},
					{
						name:     indexJS,
						url:      "http://localhost/cc/js/index.js",
						expected: indexJS,
					},
					{
						name:     styleCSS,
						url:      "http://localhost/cc/css/styles.css",
						expected: styleCSS,
					},
				}
				for _, testCase := range tests {
					t.Run(testCase.name, func(t *testing.T) {
						recorder := httptest.NewRecorder()
						request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
						helpers.NormaliseRequest(request)

						uncorsHandler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

						assert.Equal(t, 200, recorder.Code)
						assert.Equal(t, testCase.expected, testutils.ReadBody(t, recorder))
					})
				}
			})

			t.Run("should return index file by default", func(t *testing.T) {
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "http://localhost/cc/unknown.html", nil)
				helpers.NormaliseRequest(request)

				uncorsHandler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, indexHTML, testutils.ReadBody(t, recorder))
			})

			t.Run("should return error code when index file doesn't exists", func(t *testing.T) {
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "http://localhost/pnp/unknown.html", nil)
				helpers.NormaliseRequest(request)

				uncorsHandler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				expectedMessage := "filed to open index file: open /assets/index.php: file does not exist"
				assert.Contains(t, testutils.ReadBody(t, recorder), expectedMessage)
			})
		})

		t.Run("without index file", func(t *testing.T) {
			t.Run("should return static file", func(t *testing.T) {
				tests := []struct {
					name     string
					url      string
					expected string
				}{
					{
						name:     backgroundPng,
						url:      "http://localhost/img/background.png",
						expected: backgroundPng,
					},
					{
						name:     iconsSvg,
						url:      "http://localhost/img/svg/icons.svg",
						expected: iconsSvg,
					},
				}
				for _, testCase := range tests {
					t.Run(testCase.name, func(t *testing.T) {
						recorder := httptest.NewRecorder()
						request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
						helpers.NormaliseRequest(request)

						uncorsHandler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

						assert.Equal(t, http.StatusOK, recorder.Code)
						assert.Equal(t, testCase.expected, testutils.ReadBody(t, recorder))
					})
				}
			})

			t.Run("should return original file", func(t *testing.T) {
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "http://localhost/img/original.png", nil)
				helpers.NormaliseRequest(request)

				uncorsHandler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, "original.png", testutils.ReadBody(t, recorder))
			})
		})
	})

	t.Run("mocks", func(t *testing.T) {
		t.Run("should return mock with", func(t *testing.T) {
			tests := []struct {
				name         string
				url          string
				expected     string
				expectedCode int
			}{
				{
					name:         "raw content mock",
					url:          "http://localhost/api/mocks/1",
					expected:     "mock-1",
					expectedCode: http.StatusOK,
				},
				{
					name:         "file content mock",
					url:          "http://localhost/api/mocks/2",
					expected:     mockJSON,
					expectedCode: http.StatusOK,
				},
				{
					name:         "MultiStatus mock",
					url:          "http://localhost/api/mocks/3",
					expected:     "mock-3",
					expectedCode: http.StatusMultiStatus,
				},
			}
			for _, testCase := range tests {
				t.Run(testCase.name, func(t *testing.T) {
					recorder := httptest.NewRecorder()
					request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
					helpers.NormaliseRequest(request)

					uncorsHandler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

					assert.Equal(t, testCase.expectedCode, recorder.Code)
					assert.Equal(t, testCase.expected, testutils.ReadBody(t, recorder))
				})
			}
		})

		t.Run("should return error code when mock file doesn't exists", func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "http://localhost/api/mocks/4", nil)
			helpers.NormaliseRequest(request)

			uncorsHandler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

			assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			expectedMessage := "filed to open file /unknown.json: open /unknown.json: file does not exist"
			assert.Contains(t, testutils.ReadBody(t, recorder), expectedMessage)
		})
	})
}

func TestMockMiddleware(t *testing.T) {
	log.SetOutput(io.Discard)
	logger := log.New(io.Discard)

	t.Run("request method handling", func(t *testing.T) {
		t.Run("where mock method is not set allow method", func(t *testing.T) {
			requestHandler := handler.NewUncorsRequestHandler(
				handler.WithProxyHandlerFactory(proxyFactory(t, nil, nil)),
				handler.WithLogger(logger),
				handler.WithMappings(config.Mappings{
					{
						From: "*",
						To:   "*",
						Mocks: config.Mocks{
							{
								Path: "/api",
								Response: config.Response{
									Code: http.StatusOK,
									Raw:  mock1Body,
								},
							},
						},
					},
				}),
				handler.WithCacheMiddlewareFactory(cacheFactory()),
				handler.WithMockHandlerFactory(mockFactory(nil)),
				handler.WithOptionsHandlerFactory(optionsFactory()),
			)

			methods := []string{
				http.MethodGet,
				http.MethodHead,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
				http.MethodTrace,
			}

			for _, method := range methods {
				t.Run(method, func(t *testing.T) {
					request := httptest.NewRequest(method, api, nil)
					recorder := httptest.NewRecorder()

					requestHandler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

					body := testutils.ReadBody(t, recorder)
					assert.Equal(t, http.StatusOK, recorder.Code)
					assert.Equal(t, mock1Body, body)
				})
			}
		})

		t.Run("where method is set", func(t *testing.T) {
			expectedCode := 299
			expectedBody := "forwarded"
			mappings := config.Mappings{
				{From: "*", To: "*", Mocks: config.Mocks{{
					Path:   "/api",
					Method: http.MethodPut,
					Response: config.Response{
						Code: http.StatusOK,
						Raw:  mock1Body,
					},
				}}},
			}
			factory := urlreplacer.NewURLReplacerFactory(mappings)

			middleware := handler.NewUncorsRequestHandler(
				handler.WithLogger(logger),
				handler.WithMappings(mappings),
				handler.WithCacheMiddlewareFactory(cacheFactory()),
				handler.WithProxyHandlerFactory(proxyFactory(t, factory, mocks.NewHTTPClientMock(t).DoMock.
					Set(func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							Request:    req,
							StatusCode: expectedCode,
							Body:       io.NopCloser(strings.NewReader(expectedBody)),
						}, nil
					}))),
				handler.WithMockHandlerFactory(mockFactory(nil)),
				handler.WithOptionsHandlerFactory(optionsFactory()),
			)

			t.Run("method is not matched", func(t *testing.T) {
				methods := []string{
					http.MethodGet,
					http.MethodHead,
					http.MethodPost,
					http.MethodPatch,
					http.MethodDelete,
					http.MethodTrace,
				}

				for _, method := range methods {
					t.Run(method, func(t *testing.T) {
						request := httptest.NewRequest(method, api, nil)
						recorder := httptest.NewRecorder()

						middleware.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

						assert.Equal(t, expectedCode, recorder.Code)
						assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
					})
				}

				t.Run(http.MethodOptions, func(t *testing.T) {
					request := httptest.NewRequest(http.MethodOptions, api, nil)
					recorder := httptest.NewRecorder()

					middleware.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

					assert.Equal(t, http.StatusOK, recorder.Code)
					assert.Equal(t, "", testutils.ReadBody(t, recorder))
				})
			})

			t.Run("method is matched", func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPut, api, nil)
				recorder := httptest.NewRecorder()

				middleware.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

				body := testutils.ReadBody(t, recorder)
				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, mock1Body, body)
			})
		})
	})

	t.Run("path handling", func(t *testing.T) {
		expectedCode := 299
		expectedBody := "forwarded"
		mappings := config.Mappings{
			{From: "*", To: "*", Mocks: config.Mocks{
				{
					Path: userPath,
					Response: config.Response{
						Code: http.StatusOK,
						Raw:  mock1Body,
					},
				},
				{
					Path: "/api/user/{id:[0-9]+}",
					Response: config.Response{
						Code: http.StatusAccepted,
						Raw:  mock2Body,
					},
				},
				{
					Path: "/api/{single-path/demo",
					Response: config.Response{
						Code: http.StatusBadRequest,
						Raw:  mock3Body,
					},
				},
				{
					Path: `/api/v2/{multiple-path:[a-z-\/]+}/demo`,
					Response: config.Response{
						Code: http.StatusCreated,
						Raw:  mock4Body,
					},
				},
			}},
		}
		factory := urlreplacer.NewURLReplacerFactory(mappings)

		middleware := handler.NewUncorsRequestHandler(
			handler.WithLogger(logger),
			handler.WithMappings(mappings),
			handler.WithCacheMiddlewareFactory(cacheFactory()),
			handler.WithProxyHandlerFactory(proxyFactory(t, factory, mocks.NewHTTPClientMock(t).DoMock.
				Set(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						Request:    req,
						StatusCode: expectedCode,
						Body:       io.NopCloser(strings.NewReader(expectedBody)),
					}, nil
				}))),
			handler.WithMockHandlerFactory(mockFactory(nil)),
			handler.WithOptionsHandlerFactory(optionsFactory()),
		)

		tests := []struct {
			name       string
			url        string
			expected   string
			statusCode int
		}{
			{
				name:       "direct path",
				url:        apiUser,
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name:       "direct path with ending slash",
				url:        "https://localhost/api/user/",
				expected:   expectedBody,
				statusCode: expectedCode,
			},
			{
				name:       "direct path with parameter",
				url:        "https://localhost/api/user/23",
				expected:   mock2Body,
				statusCode: http.StatusAccepted,
			},
			{
				name:       "direct path with incorrect parameter",
				url:        "https://localhost/api/user/unknow",
				expected:   expectedBody,
				statusCode: expectedCode,
			},
			{
				name:       "path with subpath to single matching param",
				url:        "https://localhost/api/some-path/with-some-subpath/demo",
				expected:   expectedBody,
				statusCode: expectedCode,
			},
			{
				name:       "path with subpath to multiple matching param",
				url:        "https://localhost/api/v2/some-path/with-some-subpath/demo",
				expected:   mock4Body,
				statusCode: http.StatusCreated,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
				recorder := httptest.NewRecorder()

				middleware.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

				body := testutils.ReadBody(t, recorder)
				assert.Equal(t, testCase.statusCode, recorder.Code)
				assert.Equal(t, testCase.expected, body)
			})
		}
	})

	t.Run("query handling", func(t *testing.T) {
		middleware := handler.NewUncorsRequestHandler(
			handler.WithLogger(logger),
			handler.WithMappings(config.Mappings{
				{From: "*", To: "*", Mocks: config.Mocks{
					{
						Path: userPath,
						Response: config.Response{
							Code: http.StatusOK,
							Raw:  mock1Body,
						},
					},
					{
						Path: userPath,
						Queries: map[string]string{
							"id": "17",
						},
						Response: config.Response{
							Code: http.StatusCreated,
							Raw:  mock2Body,
						},
					},
					{
						Path: userPath,
						Queries: map[string]string{
							"id":    "99",
							"token": "000000000000000000000000000000",
						},
						Response: config.Response{
							Code: http.StatusAccepted,
							Raw:  mock3Body,
						},
					},
				}},
			}),
			handler.WithCacheMiddlewareFactory(cacheFactory()),
			handler.WithProxyHandlerFactory(proxyFactory(t, nil, nil)),
			handler.WithMockHandlerFactory(mockFactory(nil)),
			handler.WithOptionsHandlerFactory(optionsFactory()),
		)

		tests := []struct {
			name       string
			url        string
			expected   string
			statusCode int
		}{
			{
				name:       "queries is not set",
				url:        "http://localhost/api/user",
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name:       "passed unsetted parameter",
				url:        "http://localhost/api/user?id=16",
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name:       "passed parameter",
				url:        "http://localhost/api/user?id=17",
				expected:   mock2Body,
				statusCode: http.StatusCreated,
			},
			{
				name:       "passed one of multiple parameters",
				url:        "http://localhost/api/user?id=99",
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name:       "passed all of multiple parameters",
				url:        "http://localhost/api/user?id=99&token=000000000000000000000000000000",
				expected:   mock3Body,
				statusCode: http.StatusAccepted,
			},
			{
				name:       "passed extra parameters",
				url:        "http://localhost/api/user?id=99&token=000000000000000000000000000000&demo=true",
				expected:   mock3Body,
				statusCode: http.StatusAccepted,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
				recorder := httptest.NewRecorder()

				middleware.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

				body := testutils.ReadBody(t, recorder)
				assert.Equal(t, testCase.statusCode, recorder.Code)
				assert.Equal(t, testCase.expected, body)
			})
		}
	})

	t.Run("header handling", func(t *testing.T) {
		middleware := handler.NewUncorsRequestHandler(
			handler.WithLogger(logger),
			handler.WithMappings(config.Mappings{
				{From: "*", To: "*", Mocks: config.Mocks{
					{
						Path: userPath,
						Response: config.Response{
							Code: http.StatusOK,
							Raw:  mock1Body,
						},
					},
					{
						Path: userPath,
						Headers: map[string]string{
							headers.XCSRFToken: "de4e27987d054577b0edc0e828851724",
						},
						Response: config.Response{
							Code: http.StatusCreated,
							Raw:  mock2Body,
						},
					},
					{
						Path: userPath,
						Headers: map[string]string{
							userIDHeader:       "99",
							headers.XCSRFToken: "000000000000000000000000000000",
						},
						Response: config.Response{
							Code: http.StatusAccepted,
							Raw:  mock3Body,
						},
					},
				}},
			}),
			handler.WithCacheMiddlewareFactory(cacheFactory()),
			handler.WithProxyHandlerFactory(proxyFactory(t, nil, nil)),
			handler.WithMockHandlerFactory(mockFactory(nil)),
			handler.WithOptionsHandlerFactory(optionsFactory()),
		)

		tests := []struct {
			name       string
			url        string
			headers    map[string]string
			expected   string
			statusCode int
		}{
			{
				name:       "headers is not set",
				url:        apiUser,
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name: "passed unsetted headers",
				url:  apiUser,
				headers: map[string]string{
					headers.XCSRFToken: "55cc413b96026e833835a2c9a3f39c21",
				},
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name: "passed defined header",
				url:  apiUser,
				headers: map[string]string{
					headers.XCSRFToken: "de4e27987d054577b0edc0e828851724",
				},
				expected:   mock2Body,
				statusCode: http.StatusCreated,
			},
			{
				name: "passed one of multiple headers",
				url:  apiUser,
				headers: map[string]string{
					userIDHeader: "99",
				},
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name: "passed all of multiple headers",
				url:  apiUser,
				headers: map[string]string{
					userIDHeader:       "99",
					headers.XCSRFToken: "000000000000000000000000000000",
				},
				expected:   mock3Body,
				statusCode: http.StatusAccepted,
			},
			{
				name: "passed extra headers",
				url:  apiUser,
				headers: map[string]string{
					userIDHeader:           "99",
					headers.XCSRFToken:     "000000000000000000000000000000",
					headers.AcceptEncoding: "deflate",
				},
				expected:   mock3Body,
				statusCode: http.StatusAccepted,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPost, testCase.url, nil)
				for key, value := range testCase.headers {
					request.Header.Add(key, value)
				}
				recorder := httptest.NewRecorder()

				middleware.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

				body := testutils.ReadBody(t, recorder)
				assert.Equal(t, testCase.statusCode, recorder.Code)
				assert.Equal(t, testCase.expected, body)
			})
		}
	})
}
