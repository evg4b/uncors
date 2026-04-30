package cache

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/samber/lo"
)

type Middleware struct {
	output    contracts.Output
	cache     contracts.Cache
	methods   []string
	pathGlobs config.CacheGlobs
}

func NewMiddleware(options ...MiddlewareOption) *Middleware {
	middleware := helpers.ApplyOptions(&Middleware{}, options)

	helpers.AssertIsDefined(middleware.cache, "Cache storage is not configured")

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

	if cachedResponse := m.getCachedResponse(cacheKey); cachedResponse != nil {
		m.writeCachedResponse(writer, cachedResponse)
		m.output.Request(helpers.ToRequestData(request, writer.StatusCode()))

		return
	}

	cacheableWriter := NewCacheableResponseWriter(m.cache, writer, cacheKey)
	defer cacheableWriter.Close()

	next.ServeHTTP(cacheableWriter, request)
}

func (m *Middleware) writeCachedResponse(writer contracts.ResponseWriter, cachedResponse *contracts.CachedResponse) {
	header := writer.Header()

	for _, cachedHeader := range cachedResponse.Headers {
		for _, value := range cachedHeader.Value {
			header.Add(cachedHeader.Name, value)
		}
	}

	writer.WriteHeader(cachedResponse.Code)

	if len(cachedResponse.Body) > 0 {
		_, err := writer.Write(cachedResponse.Body)
		if err != nil {
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

	return fmt.Sprintf("[%s]%s%s?%s", method, url.Hostname(), url.Path, strings.Join(items, ";"))
}

func (m *Middleware) getCachedResponse(cacheKey string) *contracts.CachedResponse {
	if cachedResponse, ok := m.cache.Get(cacheKey); ok {
		return &cachedResponse
	}

	return nil
}

type MiddlewareOption = func(*Middleware)

func WithOutput(output contracts.Output) MiddlewareOption {
	return func(m *Middleware) {
		m.output = output
	}
}

func WithMethods(methods []string) MiddlewareOption {
	return func(m *Middleware) {
		m.methods = methods
	}
}

func WithGlobs(globs config.CacheGlobs) MiddlewareOption {
	return func(m *Middleware) {
		m.pathGlobs = globs
	}
}

func WithCacheStorage(cache contracts.Cache) MiddlewareOption {
	return func(m *Middleware) {
		m.cache = cache
	}
}
