package cache_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cacheProbe struct {
	handler *testutils.CountableHandler
	wrapped http.Handler
	storage *cache.RistrettoCache
}

func newProbe(t *testing.T, methods []string, respond http.HandlerFunc) *cacheProbe {
	t.Helper()

	storage := cache.NewRistrettoCache(1024*1024, time.Minute)

	t.Cleanup(func() {
		require.NoError(t, storage.Close())
	})

	handler := testutils.NewCounter(respond)

	middleware := cache.NewMiddleware(
		cache.WithCacheStorage(storage),
		cache.WithMethods(methods),
		cache.WithGlobs(config.CacheGlobs{"/api/**"}),
	)

	return &cacheProbe{handler: handler, wrapped: middleware.Wrap(handler), storage: storage}
}

func (p *cacheProbe) do(t *testing.T, request *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	p.wrapped.ServeHTTP(infra.NewResponseRecorder(recorder), request)
	p.storage.Wait()

	return recorder
}

func post(t *testing.T, path, body string) *http.Request {
	t.Helper()

	return httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, strings.NewReader(body))
}

func get(t *testing.T, path string) *http.Request {
	t.Helper()

	return httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
}

// Two POSTs to the same path used to share a cache entry, so the second request
// was answered with the first one's response.
func TestCacheKeyIncludesTheRequestBody(t *testing.T) {
	probe := newProbe(t, []string{http.MethodPost}, func(writer http.ResponseWriter, request *http.Request) {
		body := make([]byte, request.ContentLength)
		_, _ = request.Body.Read(body)

		_, _ = writer.Write([]byte("answer for " + string(body)))
	})

	first := probe.do(t, post(t, "/api/search", "alpha"))
	second := probe.do(t, post(t, "/api/search", "beta"))
	repeat := probe.do(t, post(t, "/api/search", "alpha"))

	assert.Equal(t, "answer for alpha", first.Body.String())
	assert.Equal(t, "answer for beta", second.Body.String())
	assert.Equal(t, "answer for alpha", repeat.Body.String())
	assert.Equal(t, 2, probe.handler.Count(), "only the repeated body is served from the cache")
}

func TestCacheKeyIncludesThePort(t *testing.T) {
	probe := newProbe(t, []string{http.MethodGet}, func(writer http.ResponseWriter, request *http.Request) {
		//nolint:gosec // G705: a test handler echoing the host it was called on
		_, _ = writer.Write([]byte("from " + request.Host))
	})

	first := probe.do(t, get(t, "http://api.local:3000/api/data"))
	second := probe.do(t, get(t, "http://api.local:4000/api/data"))

	assert.Equal(t, "from api.local:3000", first.Body.String())
	assert.Equal(t, "from api.local:4000", second.Body.String())
	assert.Equal(t, 2, probe.handler.Count(), "mappings on different ports are different resources")
}

func TestCacheRespectsResponseDirectives(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
		cached bool
	}{
		{name: "plain response", header: http.Header{}, cached: true},
		{name: "no-store", header: http.Header{headers.CacheControl: {"no-store"}}, cached: false},
		{name: "private", header: http.Header{headers.CacheControl: {"private, max-age=60"}}, cached: false},
		{name: "set-cookie", header: http.Header{headers.SetCookie: {"session=abc"}}, cached: false},
		{name: "vary on accept-encoding", header: http.Header{headers.Vary: {"Accept-Encoding"}}, cached: true},
		{name: "vary on anything else", header: http.Header{headers.Vary: {"Accept-Language"}}, cached: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			probe := newProbe(t, []string{http.MethodGet}, func(writer http.ResponseWriter, _ *http.Request) {
				testutils.CopyHeaders(testCase.header, writer.Header())
				writer.WriteHeader(http.StatusOK)
				_, _ = writer.Write([]byte("body"))
			})

			probe.do(t, get(t, "/api/data"))
			probe.do(t, get(t, "/api/data"))

			expected := 2
			if testCase.cached {
				expected = 1
			}

			assert.Equal(t, expected, probe.handler.Count())
		})
	}
}

// Replaying an authenticated response to whoever asks next is a data leak.
func TestCacheSkipsAuthenticatedRequests(t *testing.T) {
	probe := newProbe(t, []string{http.MethodGet}, func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("secret"))
	})

	authorised := get(t, "/api/profile")
	authorised.Header.Set(headers.Authorization, "Bearer token")

	probe.do(t, authorised)
	probe.do(t, get(t, "/api/profile"))

	assert.Equal(t, 2, probe.handler.Count())
}
