package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func rewriteFactory() handler.RewriteMiddlewareFactory {
	return func(rewriting config.RewritingOption) contracts.Middleware {
		return rewrite.NewMiddleware(rewrite.WithRewritingOptions(rewriting))
	}
}

func scriptHandlerFactory() handler.ScriptHandlerFactory {
	return func(_ config.Script) contracts.Handler {
		return contracts.HandlerFunc(func(w contracts.ResponseWriter, _ *contracts.Request) {
			w.WriteHeader(http.StatusAccepted)
		})
	}
}

func TestHandlerWithOutput(t *testing.T) {
	t.Run("logs and returns error for unmapped host", func(t *testing.T) {
		outputMock := mocks.NewOutputMock(t)
		outputMock.ErrorfMock.Set(func(_ string, _ ...any) {})

		requestHandler := handler.NewUncorsRequestHandler(
			handler.WithMappings(config.Mappings{
				{From: "http://mapped.host", To: "https://mapped.host"},
			}),
			handler.WithCacheMiddlewareFactory(cacheFactory()),
			handler.WithProxyHandler(proxyFactory(t, nil, nil)),
			handler.WithMockHandlerFactory(mockFactory(nil)),
			handler.WithOptionsHandlerFactory(optionsFactory()),
			handler.WithOutput(outputMock),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://unmapped.host/api", nil)
		helpers.NormaliseRequest(request)

		requestHandler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestHandlerWithScripts(t *testing.T) {
	t.Run("routes request to script handler", func(t *testing.T) {
		requestHandler := handler.NewUncorsRequestHandler(
			handler.WithMappings(config.Mappings{
				{
					From: "{host}",
					To:   "{host}",
					Scripts: config.Scripts{
						{
							Matcher: config.RequestMatcher{Path: "/script"},
							Script:  `response:send("ok")`,
						},
					},
				},
			}),
			handler.WithCacheMiddlewareFactory(cacheFactory()),
			handler.WithProxyHandler(proxyFactory(t, nil, nil)),
			handler.WithMockHandlerFactory(mockFactory(nil)),
			handler.WithOptionsHandlerFactory(optionsFactory()),
			handler.WithScriptHandlerFactory(scriptHandlerFactory()),
			handler.WithOutput(mocks.NoopOutput()),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/script", nil)

		requestHandler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

		assert.Equal(t, http.StatusAccepted, recorder.Code)
	})
}

func TestHandlerWithRewrites(t *testing.T) {
	fs := testutils.FsFromMap(t, map[string]string{})

	t.Run("rewrites request path and proxies", func(t *testing.T) {
		const rewrittenPath = "/api/v2/resource"

		httpMock := mocks.NewHTTPClientMock(t).DoMock.Set(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, rewrittenPath, req.URL.Path)

			return &http.Response{
				Body:       http.NoBody,
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Request:    req,
			}, nil
		})

		mappings := config.Mappings{
			{
				From: "http://localhost",
				To:   "https://localhost",
				Rewrites: config.RewriteOptions{
					{From: "/api/v1/resource", To: rewrittenPath},
				},
			},
		}

		requestHandler := handler.NewUncorsRequestHandler(
			handler.WithMappings(mappings),
			handler.WithCacheMiddlewareFactory(cacheFactory()),
			handler.WithProxyHandler(proxyFactory(t, urlreplacer.NewURLReplacerFactory(mappings), httpMock)),
			handler.WithStaticHandlerFactory(staticFactory(fs)),
			handler.WithMockHandlerFactory(mockFactory(nil)),
			handler.WithOptionsHandlerFactory(optionsFactory()),
			handler.WithRewriteHandlerFactory(rewriteFactory()),
			handler.WithOutput(mocks.NoopOutput()),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/api/v1/resource", nil)
		helpers.NormaliseRequest(request)

		requestHandler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestHandlerWithCache(t *testing.T) {
	t.Run("proxies requests when cache globs are configured", func(t *testing.T) {
		httpMock := mocks.NewHTTPClientMock(t).DoMock.Set(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Body:       http.NoBody,
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Request:    req,
			}, nil
		})

		mappings := config.Mappings{
			{
				From:  "http://localhost",
				To:    "https://localhost",
				Cache: config.CacheGlobs{"/api/*"},
			},
		}

		requestHandler := handler.NewUncorsRequestHandler(
			handler.WithMappings(mappings),
			handler.WithCacheMiddlewareFactory(cacheFactory()),
			handler.WithProxyHandler(proxyFactory(t, urlreplacer.NewURLReplacerFactory(mappings), httpMock)),
			handler.WithMockHandlerFactory(mockFactory(nil)),
			handler.WithOptionsHandlerFactory(optionsFactory()),
			handler.WithOutput(mocks.NoopOutput()),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/api/data", nil)
		helpers.NormaliseRequest(request)
		requestHandler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestHandlerWithOptionsDisabled(t *testing.T) {
	t.Run("forwards OPTIONS requests when options middleware is disabled", func(t *testing.T) {
		httpMock := mocks.NewHTTPClientMock(t).DoMock.Set(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Body:       http.NoBody,
				StatusCode: http.StatusNoContent,
				Header:     http.Header{},
				Request:    req,
			}, nil
		})

		mappings := config.Mappings{
			{
				From:            "http://localhost",
				To:              "https://localhost",
				OptionsHandling: config.OptionsHandling{Disabled: true},
			},
		}

		requestHandler := handler.NewUncorsRequestHandler(
			handler.WithMappings(mappings),
			handler.WithCacheMiddlewareFactory(cacheFactory()),
			handler.WithProxyHandler(proxyFactory(t, urlreplacer.NewURLReplacerFactory(mappings), httpMock)),
			handler.WithMockHandlerFactory(mockFactory(nil)),
			handler.WithOptionsHandlerFactory(optionsFactory()),
			handler.WithOutput(mocks.NoopOutput()),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "http://localhost/api", nil)
		helpers.NormaliseRequest(request)

		requestHandler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

		assert.Equal(t, http.StatusNoContent, recorder.Code)
	})
}
