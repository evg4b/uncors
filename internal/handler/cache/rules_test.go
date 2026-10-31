package cache_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/headers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cachedURL = "http://localhost/api/data"

type cachedServer struct {
	handler contracts.Handler
	store   *cache.RistrettoCache
	counter *testutils.CountableHandler
}

func newCachedServer(t *testing.T, serve func(contracts.ResponseWriter, *contracts.Request) error) *cachedServer {
	t.Helper()

	store := lo.Must(cache.NewRistrettoCache(1024*1024, time.Minute))

	t.Cleanup(func() { testutils.Close(t, store) })

	counter := testutils.NewCounter(serve)
	middleware := cache.NewMiddleware(
		cache.WithCacheStorage(store),
		cache.WithMethods([]string{http.MethodGet}),
		cache.WithGlobs(config.CacheGlobs{"/api/**"}),
	)

	return &cachedServer{
		handler: infra.Wrap(middleware, counter),
		store:   store,
		counter: counter,
	}
}

func (s *cachedServer) serve(t *testing.T, url string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	testutils.CopyHeaders(header, request.Header)

	require.NoError(t, s.handler.ServeHTTP(server.NewResponseRecorder(recorder), request))
	s.store.Wait()

	return recorder
}

func TestCacheMiddlewareCorsHeaders(t *testing.T) {
	cached := newCachedServer(t, proxyLike(http.StatusOK, http.Header{
		headers.ContentType: {"application/json"},
		"X-Request-Id":      {"first"},
	}, "body"))

	first := cached.serve(t, cachedURL, http.Header{headers.Origin: {"http://first.local"}})
	assert.Equal(t, "http://first.local", first.Header().Get(headers.AccessControlAllowOrigin))

	tests := []struct {
		name   string
		origin string
	}{
		{name: "another origin", origin: "http://second.local"},
		{name: "no origin", origin: ""},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			header := http.Header{}
			if testCase.origin != "" {
				header.Set(headers.Origin, testCase.origin)
			}

			recorder := cached.serve(t, cachedURL, header)

			assert.Equal(t, 1, cached.counter.Count())
			assert.Equal(t, "body", testutils.ReadBody(t, recorder))
			assert.Equal(t, withCors(http.Header{
				headers.ContentType: {"application/json"},
				"X-Request-Id":      {"first"},
				headers.Vary:        {headers.Origin},
				"Age":               {"0"},
			}, testCase.origin), recorder.Header())
		})
	}
}

func TestCacheMiddlewareStoredHeaders(t *testing.T) {
	var stored contracts.CachedResponse

	storage := mocks.NewCacheMock(t).
		GetMock.Return(contracts.CachedResponse{}, false).
		SetMock.Set(func(_ string, value contracts.CachedResponse) bool {
		stored = value

		return true
	})

	middleware := cache.NewMiddleware(
		cache.WithCacheStorage(storage),
		cache.WithMethods([]string{http.MethodGet}),
		cache.WithGlobs(config.CacheGlobs{"/api/**"}),
	)

	handler := infra.Wrap(middleware, infra.HandlerFunc(
		proxyLike(http.StatusOK, http.Header{"X-Custom": {"value"}}, "body"),
	))

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, cachedURL, nil)
	request.Header.Set(headers.Origin, "http://first.local")
	require.NoError(t, handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), request))

	assert.Equal(t, []contracts.CachedHeader{
		testutils.CachedHeader(headers.Vary, headers.Origin),
		testutils.CachedHeader("X-Custom", "value"),
	}, stored.Headers)
}

func TestCacheMiddlewareSkipsResponses(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
	}{
		{name: "with Set-Cookie", header: http.Header{headers.SetCookie: {"session=1"}}},
		{name: "with Cache-Control no-store", header: http.Header{headers.CacheControl: {"no-store"}}},
		{name: "with Cache-Control private", header: http.Header{headers.CacheControl: {"max-age=60, private"}}},
		{name: "with Cache-Control private fields", header: http.Header{headers.CacheControl: {`Private="X-User"`}}},
		{name: "with Vary *", header: http.Header{headers.Vary: {"Origin, *"}}},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			cached := newCachedServer(t, proxyLike(http.StatusOK, testCase.header, "body"))

			testutils.Times(3, func(_ int) {
				recorder := cached.serve(t, cachedURL, nil)
				assert.Equal(t, "body", testutils.ReadBody(t, recorder))
			})

			assert.Equal(t, 3, cached.counter.Count())
		})
	}

	t.Run("stores a public response", func(t *testing.T) {
		cached := newCachedServer(t, proxyLike(http.StatusOK, http.Header{
			headers.CacheControl: {"public, max-age=60"},
		}, "body"))

		testutils.Times(3, func(_ int) {
			cached.serve(t, cachedURL, nil)
		})

		assert.Equal(t, 1, cached.counter.Count())
	})
}

func TestCacheMiddlewareAcceptEncoding(t *testing.T) {
	cached := newCachedServer(t, func(writer contracts.ResponseWriter, request *contracts.Request) error {
		fmt.Fprint(writer, request.Header.Get(headers.AcceptEncoding))

		return nil
	})

	tests := []struct {
		encoding string
		body     string
		calls    int
	}{
		{encoding: "gzip, br", body: "gzip, br", calls: 1},
		{encoding: "", body: "", calls: 2},
		{encoding: "BR,gzip", body: "gzip, br", calls: 2},
		{encoding: "identity", body: "identity", calls: 3},
		{encoding: "", body: "", calls: 3},
	}

	for _, testCase := range tests {
		header := http.Header{}
		if testCase.encoding != "" {
			header.Set(headers.AcceptEncoding, testCase.encoding)
		}

		recorder := cached.serve(t, cachedURL, header)

		assert.Equal(t, testCase.body, testutils.ReadBody(t, recorder), testCase.encoding)
		assert.Equal(t, testCase.calls, cached.counter.Count(), testCase.encoding)
	}
}

func TestCacheMiddlewareVary(t *testing.T) {
	cached := newCachedServer(t, func(writer contracts.ResponseWriter, request *contracts.Request) error {
		writer.Header().Set(headers.Vary, "accept-language, Origin")
		fmt.Fprint(writer, request.Header.Get(headers.AcceptLanguage))

		return nil
	})

	tests := []struct {
		language string
		origin   string
		calls    int
	}{
		{language: "en", origin: "http://first.local", calls: 1},
		{language: "fr", origin: "http://first.local", calls: 2},
		{language: "en", origin: "http://second.local", calls: 2},
		{language: "fr", calls: 2},
		{language: "", calls: 3},
	}

	for _, testCase := range tests {
		header := http.Header{}
		if testCase.language != "" {
			header.Set(headers.AcceptLanguage, testCase.language)
		}

		if testCase.origin != "" {
			header.Set(headers.Origin, testCase.origin)
		}

		recorder := cached.serve(t, cachedURL, header)

		assert.Equal(t, testCase.language, testutils.ReadBody(t, recorder), testCase.language)
		assert.Equal(t, testCase.calls, cached.counter.Count(), testCase.language)
	}
}

func TestCacheMiddlewareQueryOrder(t *testing.T) {
	cached := newCachedServer(t, func(writer contracts.ResponseWriter, request *contracts.Request) error {
		fmt.Fprint(writer, request.URL.RawQuery)

		return nil
	})

	tests := []struct {
		query string
		body  string
		calls int
	}{
		{query: "a=2&a=1", body: "a=2&a=1", calls: 1},
		{query: "a=1&a=2", body: "a=1&a=2", calls: 2},
		{query: "a=1,2", body: "a=1,2", calls: 3},
		{query: "b=1&a=2&a=1", body: "b=1&a=2&a=1", calls: 4},
		{query: "a=2&b=1&a=1", body: "b=1&a=2&a=1", calls: 4},
		{query: "a=2&a=1", body: "a=2&a=1", calls: 4},
	}

	for _, testCase := range tests {
		recorder := cached.serve(t, cachedURL+"?"+testCase.query, nil)

		assert.Equal(t, testCase.body, testutils.ReadBody(t, recorder), testCase.query)
		assert.Equal(t, testCase.calls, cached.counter.Count(), testCase.query)
	}
}
