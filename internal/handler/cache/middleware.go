package cache

import (
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/patrickmn/go-cache"
	"github.com/samber/lo"
)

type Middleware struct {
	logger    contracts.Logger
	storage   *cache.Cache
	prefix    string
	methods   []string
	pathGlobs config.CacheGlobs
}

func NewMiddleware(options ...MiddlewareOption) *Middleware {
	middleware := &Middleware{}

	for _, option := range options {
		option(middleware)
	}

	helpers.AssertIsDefined(middleware.logger, "Logger is not configured")
	helpers.AssertIsDefined(middleware.storage, "Cache storage is not configured")

	return middleware
}

func (m *Middleware) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) {
		if !m.cacheableRequest(request) {
			next.ServeHTTP(writer, request)

			return
		}

		cacheKey := m.extractKey(request.URL)
		if cachedResponse := m.getCachedResponse(cacheKey); cachedResponse != nil {
			m.writeCachedResponse(writer, cachedResponse)

			m.logger.PrintResponse(request, writer.StatusCode())

			return
		}

		cacheableWriter := NewCacheableWriter(writer)
		next.ServeHTTP(cacheableWriter, request)
		if helpers.Is2xxCode(cacheableWriter.StatusCode()) {
			response := cacheableWriter.GetCachedResponse()
			m.storage.Set(cacheKey, response, time.Hour)
		}
	})
}

func (m *Middleware) writeCachedResponse(writer contracts.ResponseWriter, cachedResponse *CachedResponse) {
	header := writer.Header()
	for key, values := range cachedResponse.Header {
		for _, value := range values {
			header.Add(key, value)
		}
	}

	writer.WriteHeader(cachedResponse.Code)
	if cachedResponse.Body != nil && len(cachedResponse.Body) > 0 {
		if _, err := writer.Write(cachedResponse.Body); err != nil {
			panic(err)
		}
	}
}

func (m *Middleware) cacheableRequest(request *contracts.Request) bool {
	return lo.Contains(m.methods, request.Method) && lo.ContainsBy(m.pathGlobs, func(pattern string) bool {
		ok, err := doublestar.PathMatch(pattern, request.URL.Path)
		if err != nil {
			panic(err)
		}

		return ok
	})
}

func (m *Middleware) extractKey(url *url.URL) string {
	values := url.Query()
	items := make([]string, 0, len(values))
	for key, value := range values {
		sort.Strings(value)
		items = append(items, key+"="+strings.Join(value, ","))
	}

	sort.Strings(items)

	return m.prefix + url.Path + "?" + strings.Join(items, ";")
}

func (m *Middleware) getCachedResponse(cacheKey string) *CachedResponse {
	if cachedResponse, ok := m.storage.Get(cacheKey); ok {
		return cachedResponse.(*CachedResponse) // nolint: forcetypeassert
	}

	return nil
}
