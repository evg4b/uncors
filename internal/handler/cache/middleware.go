package cache

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/patrickmn/go-cache"
	"net/http"
	"time"
)

type Middleware struct {
	logger contracts.Logger
	cache  *cache.Cache
	prefix string
}

func NewMiddleware(options ...MiddlewareOption) *Middleware {
	middleware := &Middleware{
		cache: cache.New(time.Hour, time.Hour),
	}

	for _, option := range options {
		option(middleware)
	}

	helpers.AssertIsDefined(middleware.logger, "Logger is not configured")
	helpers.AssertIsDefined(middleware.cache, "Cache is not configured")

	return middleware
}

func (m *Middleware) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(writer *contracts.ResponseWriter, request *contracts.Request) {
		if m.cacheableRequest(request) {
			cacheKey := m.extractKey(request)
			cachedResponse, ok := m.cache.Get(cacheKey)
			if ok {

				d := cachedResponse.(*CachedResponse)

				hh := writer.Header()
				for kay, value := range d.Header {
					for _, s := range value {
						hh.Add(kay, s)
					}
				}

				writer.WriteHeader(d.Code)
				writer.Write(d.Body)

				m.logger.PrintResponse(&http.Response{
					Request:    request,
					StatusCode: writer.StatusCode,
				})

				return
			}

			wrapped := NewCacheableWriter(writer)

			next.ServeHTTP(contracts.WrapResponseWriter(wrapped), request)

			response := wrapped.GetCachedResponse()
			m.cache.Set(cacheKey, response, time.Hour)

			return
		}

		next.ServeHTTP(writer, request)
	})
}

func (m *Middleware) cacheableRequest(request *contracts.Request) bool {
	return request.Method == http.MethodGet
}

func (m *Middleware) extractKey(request *contracts.Request) string {
	return m.prefix + request.URL.Path
}
