package cache

import (
	"net/url"
	"sort"
	"strings"

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
	methods   []string
	pathGlobs config.CacheGlobs
}

func NewMiddleware(options ...MiddlewareOption) *Middleware {
	middleware := helpers.ApplyOptions(&Middleware{}, options)

	helpers.AssertIsDefined(middleware.logger, "Logger is not configured")
	helpers.AssertIsDefined(middleware.storage, "Cache storage is not configured")

	return middleware
}

func (m *Middleware) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) {
		if !m.isCacheableRequest(request) {
			next.ServeHTTP(writer, request)

			return
		}

		m.cacheRequest(writer, request, next)
	})
}

func (m *Middleware) cacheRequest(writer contracts.ResponseWriter, request *contracts.Request, next contracts.Handler) {
	cacheKey := m.extractCacheKey(request.Method, request.URL)
	m.logger.Debugf("extracted %s from request", cacheKey)
	if cachedResponse := m.getCachedResponse(cacheKey); cachedResponse != nil {
		m.logger.Debugf("extracted %s from request", cacheKey)

		m.writeCachedResponse(writer, cachedResponse)


		return
	}

	m.logger.Debugf("request with key %s is not cached", cacheKey)

	cacheableWriter := NewCacheableWriter(writer)
	next.ServeHTTP(cacheableWriter, request)
	if helpers.Is2xxCode(cacheableWriter.StatusCode()) {
		response := cacheableWriter.GetCachedResponse()
		m.storage.SetDefault(cacheKey, response)
	}
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

func (m *Middleware) isCacheableRequest(request *contracts.Request) bool {
	return lo.Contains(m.methods, request.Method) && lo.ContainsBy(m.pathGlobs, func(pattern string) bool {
		ok, err := doublestar.PathMatch(pattern, request.URL.Path)
		if err != nil {
			panic(err)
		}

		return ok
	})
}

func (m *Middleware) extractCacheKey(method string, url *url.URL) string {
	values := url.Query()
	items := make([]string, 0, len(values))
	for key, value := range values {
		sort.Strings(value)
		valuesKey := key + "=" + strings.Join(value, ",")
		items = append(items, valuesKey)
	}

	sort.Strings(items)

	return helpers.Sprintf("[%s]%s%s?%s", method, url.Hostname(), url.Path, strings.Join(items, ";"))
}

func (m *Middleware) getCachedResponse(cacheKey string) *CachedResponse {
	if cachedResponse, ok := m.storage.Get(cacheKey); ok {
		return cachedResponse.(*CachedResponse) // nolint: forcetypeassert
	}

	return nil
}
