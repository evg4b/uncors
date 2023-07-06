package cache

import (
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/go-http-utils/headers"
	"github.com/samber/lo"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/patrickmn/go-cache"
)

type Middleware struct {
	logger    contracts.Logger
	cache     *cache.Cache
	prefix    string
	methods   []string
	pathGlobs []string
}

func NewMiddleware(options ...MiddlewareOption) *Middleware {
	middleware := &Middleware{
		cache:   cache.New(time.Hour, time.Hour),
		methods: []string{http.MethodGet},
	}

	for _, option := range options {
		option(middleware)
	}

	helpers.AssertIsDefined(middleware.logger, "Logger is not configured")
	helpers.AssertIsDefined(middleware.cache, "Cache is not configured")

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

			m.logger.PrintResponse(&http.Response{
				Request:    request,
				StatusCode: writer.StatusCode(),
			})

			return
		}

		cacheableWriter := NewCacheableWriter(writer)
		next.ServeHTTP(cacheableWriter, request)

		response := cacheableWriter.GetCachedResponse()
		m.cache.Set(cacheKey, response, time.Hour)
	})
}

func (m *Middleware) writeCachedResponse(writer contracts.ResponseWriter, cachedResponse *CachedResponse) {
	header := writer.Header()
	for key, values := range cachedResponse.Header {
		if strings.EqualFold(key, headers.ContentLength) {
			for _, value := range values {
				header.Add(key, value)
			}
		}
	}

	writer.WriteHeader(cachedResponse.Code)
	if cachedResponse.Body != nil {
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
	var items []string // nolint: prealloc
	for key, value := range url.Query() {
		sort.Strings(value)
		items = append(items, key+"="+strings.Join(value, ","))
	}

	sort.Strings(items)

	return m.prefix + url.Path + "?" + strings.Join(items, ";")
}

func (m *Middleware) getCachedResponse(cacheKey string) *CachedResponse {
	if cachedResponse, ok := m.cache.Get(cacheKey); ok {
		return cachedResponse.(*CachedResponse) // nolint: forcetypeassert
	}

	return nil
}
