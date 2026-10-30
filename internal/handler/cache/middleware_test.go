package cache_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errUpstreamReset = errors.New("upstream reset")

func TestCacheMiddleware(t *testing.T) {
	const (
		expectedBody     = "this is test"
		cacheGlob        = "/api/**"
		constantEndpoint = "/api/constants"
	)

	responseHeader := http.Header{
		headers.ContentType:     {"text/html; charset=iso-8859-1"},
		headers.ContentEncoding: {"deflate, gzip"},
	}

	expectedHeader := withCors(responseHeader, "")

	store := lo.Must(cache.NewRistrettoCache(1024*1024, time.Minute))
	middleware := cache.NewMiddleware(
		cache.WithCacheStorage(store),
		cache.WithMethods([]string{http.MethodGet}),
		cache.WithGlobs(config.CacheGlobs{
			"/translations",
			cacheGlob,
		}),
	)

	testHandler := testutils.NewCounter(proxyLike(http.StatusOK, responseHeader, expectedBody))

	serveFiveTimes := func(t *testing.T, handler contracts.Handler, method, path string) {
		t.Helper()

		testutils.Times(5, func(_ int) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), method, path, nil)
			require.NoError(t, handler.ServeHTTP(server.NewResponseRecorder(recorder), request))
			store.Wait()

			header := recorder.Header().Clone()
			header.Del("Age")
			assert.Equal(t, expectedHeader, header)
			assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
		})
	}

	t.Run("should not call cached response just one time for", func(t *testing.T) {
		tests := []struct {
			name       string
			method     string
			path       string
			statusCode int
		}{
			{
				name:       "request in glob",
				method:     http.MethodGet,
				path:       "/api",
				statusCode: http.StatusOK,
			},
			{
				name:       "request in glob with params",
				method:     http.MethodGet,
				path:       "/api?some=params",
				statusCode: http.StatusOK,
			},
			{
				name:       "request in glob with other params",
				method:     http.MethodGet,
				path:       "/api?other=params",
				statusCode: http.StatusOK,
			},
			{
				name:       "second level request in glob",
				method:     http.MethodGet,
				path:       "/api/comments",
				statusCode: http.StatusOK,
			},
			{
				name:       "second level request in glob with params",
				method:     http.MethodGet,
				path:       "/api/comments?q=test",
				statusCode: http.StatusOK,
			},
			{
				name:       "third level request in glob",
				method:     http.MethodGet,
				path:       "/api/comments/1",
				statusCode: http.StatusOK,
			},
			{
				name:       "third level request in glob with params",
				method:     http.MethodGet,
				path:       "/api/comments/1?q=demo",
				statusCode: http.StatusOK,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				testHandler.Reset()

				wrappedHandler := infra.Wrap(middleware, testHandler)

				serveFiveTimes(t, wrappedHandler, testCase.method, testCase.path)

				assert.Equal(t, 1, testHandler.Count())
			})
		}
	})

	t.Run("should not cache response", func(t *testing.T) {
		tests := []struct {
			name       string
			method     string
			path       string
			statusCode int
		}{
			{
				name:       "witch path out of glob",
				method:     http.MethodGet,
				path:       "/test",
				statusCode: http.StatusOK,
			},
			{
				name:       "from POST method request",
				method:     http.MethodPost,
				path:       "/api",
				statusCode: http.StatusOK,
			},
			{
				name:       "witch response with status code 500",
				method:     http.MethodGet,
				path:       constantEndpoint,
				statusCode: http.StatusInternalServerError,
			},
			{
				name:       "witch response with status code 400",
				method:     http.MethodGet,
				path:       constantEndpoint,
				statusCode: http.StatusBadRequest,
			},
			{
				name:       "witch response with status code 304",
				method:     http.MethodGet,
				path:       constantEndpoint,
				statusCode: http.StatusNotModified,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				counter := testutils.NewCounter(proxyLike(testCase.statusCode, responseHeader, expectedBody))

				wrappedHandler := infra.Wrap(middleware, counter)

				serveFiveTimes(t, wrappedHandler, testCase.method, testCase.path)

				assert.Equal(t, 5, counter.Count())
			})
		}
	})

	t.Run("should not cache response between different hosts matched by one rule", func(t *testing.T) {
		const count = 5

		testHandler.Reset()

		middleware := cache.NewMiddleware(
			cache.WithCacheStorage(lo.Must(cache.NewRistrettoCache(1024*1024, time.Minute))),
			cache.WithMethods([]string{http.MethodGet}),
			cache.WithGlobs(config.CacheGlobs{cacheGlob}),
		)

		wrappedHandler := infra.Wrap(middleware, testHandler)

		testutils.Times(count, func(index int) {
			recorder := httptest.NewRecorder()
			rec := server.NewResponseRecorder(recorder)
			url := fmt.Sprintf("https://test-host-%d.com:4200/api/test", index)
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
			require.NoError(t, wrappedHandler.ServeHTTP(rec, request))
			assert.Equal(t, expectedHeader, recorder.Header())
			assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
		})

		assert.Equal(t, count, testHandler.Count())
	})

	t.Run("should not share cached responses between ports of one host", func(t *testing.T) {
		middleware := cache.NewMiddleware(
			cache.WithCacheStorage(lo.Must(cache.NewRistrettoCache(1024*1024, time.Minute))),
			cache.WithMethods([]string{http.MethodGet}),
			cache.WithGlobs(config.CacheGlobs{cacheGlob}),
		)

		handler := infra.Wrap(middleware, infra.HandlerFunc(
			func(writer contracts.ResponseWriter, request *contracts.Request) error {
				fmt.Fprint(writer, request.URL.String())

				return nil
			},
		))

		for _, url := range []string{
			"http://localhost:3000/api/test",
			"http://localhost:4000/api/test",
			"https://localhost:4000/api/test",
		} {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
			require.NoError(t, handler.ServeHTTP(server.NewResponseRecorder(recorder), request))
			assert.Equal(t, url, testutils.ReadBody(t, recorder))
		}
	})

	t.Run("should not cache an interrupted response", func(t *testing.T) {
		tests := []struct {
			name    string
			url     string
			wantErr error
			serve   func(writer contracts.ResponseWriter, cancel context.CancelFunc) error
		}{
			{
				name:    "handler failed",
				url:     "https://test-host.com:4200/api/partial",
				wantErr: errUpstreamReset,
				serve: func(writer contracts.ResponseWriter, _ context.CancelFunc) error {
					writer.WriteHeader(http.StatusOK)
					fmt.Fprint(writer, "partial")

					return errUpstreamReset
				},
			},
			{
				name: "request cancelled",
				url:  "https://test-host.com:4200/api/cancelled",
				serve: func(writer contracts.ResponseWriter, cancel context.CancelFunc) error {
					fmt.Fprint(writer, "partial")
					cancel()

					return nil
				},
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				middleware := cache.NewMiddleware(
					cache.WithCacheStorage(lo.Must(cache.NewRistrettoCache(1024*1024, time.Minute))),
					cache.WithMethods([]string{http.MethodGet}),
					cache.WithGlobs(config.CacheGlobs{cacheGlob}),
				)

				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()

				interrupted := infra.Wrap(middleware, infra.HandlerFunc(
					func(writer contracts.ResponseWriter, _ *contracts.Request) error {
						return testCase.serve(writer, cancel)
					},
				))

				request := httptest.NewRequestWithContext(ctx, http.MethodGet, testCase.url, nil)

				err := interrupted.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), request)
				if testCase.wantErr != nil {
					require.ErrorIs(t, err, testCase.wantErr)
				} else {
					require.NoError(t, err)
				}

				testHandler.Reset()

				recorder := httptest.NewRecorder()
				request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, testCase.url, nil)
				require.NoError(t, infra.Wrap(middleware, testHandler).
					ServeHTTP(server.NewResponseRecorder(recorder), request))
				assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
				assert.Equal(t, 1, testHandler.Count())
			})
		}
	})

	t.Run("should not cache response between different methods matched by one rule", func(t *testing.T) {
		methods := []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodPut}

		testHandler.Reset()

		middleware := cache.NewMiddleware(
			cache.WithCacheStorage(lo.Must(cache.NewRistrettoCache(1024*1024, time.Minute))),
			cache.WithMethods(methods),
			cache.WithGlobs(config.CacheGlobs{cacheGlob}),
		)

		testHandler := testutils.NewCounter(func(writer contracts.ResponseWriter, request *contracts.Request) error {
			testutils.CopyHeaders(responseHeader, writer.Header())
			infra.WriteCorsHeaders(writer.Header(), request.Header.Get(headers.Origin))
			writer.WriteHeader(http.StatusOK)
			fmt.Fprint(writer, request.Method)

			return nil
		})

		wrappedHandler := infra.Wrap(middleware, testHandler)

		for _, method := range methods {
			recorder := httptest.NewRecorder()
			rec := server.NewResponseRecorder(recorder)
			request := httptest.NewRequestWithContext(t.Context(), method, "https://test-host.com:4200/api/test", nil)
			require.NoError(t, wrappedHandler.ServeHTTP(rec, request))
			assert.Equal(t, expectedHeader, recorder.Header())
			assert.Equal(t, method, testutils.ReadBody(t, recorder))
		}

		assert.Len(t, methods, testHandler.Count())
	})
}

func TestCacheMiddlewareTruncatedCapture(t *testing.T) {
	tests := []struct {
		name   string
		size   int
		stored bool
	}{
		{name: "stores a body within the capture limit", size: 1024, stored: true},
		{name: "does not store a body over the capture limit", size: 11 << 20, stored: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			body := bytes.Repeat([]byte("x"), testCase.size)

			storage := mocks.NewCacheMock(t).
				GetMock.Return(contracts.CachedResponse{}, false)
			if testCase.stored {
				storage.SetMock.Set(func(_ string, value contracts.CachedResponse) bool {
					assert.Equal(t, body, value.Body)

					return true
				})
			}

			middleware := cache.NewMiddleware(
				cache.WithCacheStorage(storage),
				cache.WithMethods([]string{http.MethodGet}),
				cache.WithGlobs(config.CacheGlobs{"/**"}),
			)

			handler := infra.Wrap(middleware, infra.HandlerFunc(
				func(writer contracts.ResponseWriter, _ *contracts.Request) error {
					writer.WriteHeader(http.StatusOK)
					_, err := writer.Write(body)

					return err
				},
			))

			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/stream", nil)
			require.NoError(t, handler.ServeHTTP(server.NewResponseRecorder(recorder), request))

			assert.Equal(t, testCase.size, recorder.Body.Len())
		})
	}
}

func withCors(header http.Header, origin string) http.Header {
	result := header.Clone()
	infra.WriteCorsHeaders(result, origin)

	return result
}

func proxyLike(status int, header http.Header, body string) func(contracts.ResponseWriter, *contracts.Request) error {
	return func(writer contracts.ResponseWriter, request *contracts.Request) error {
		testutils.CopyHeaders(header, writer.Header())
		infra.WriteCorsHeaders(writer.Header(), request.Header.Get(headers.Origin))
		writer.WriteHeader(status)
		fmt.Fprint(writer, body)

		return nil
	}
}

func TestCacheMiddlewareAge(t *testing.T) {
	tests := []struct {
		name     string
		stored   time.Duration
		upstream string
		want     string
	}{
		{name: "counts the time in the cache", stored: 90 * time.Second, want: "90"},
		{name: "adds the upstream age", stored: 90 * time.Second, upstream: "10", want: "100"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			stored := []contracts.CachedHeader{}
			if testCase.upstream != "" {
				stored = append(stored, contracts.CachedHeader{Name: "Age", Value: []string{testCase.upstream}})
			}

			storage := mocks.NewCacheMock(t).GetMock.Return(contracts.CachedResponse{
				Code:    http.StatusOK,
				Headers: stored,
				Stored:  time.Now().Add(-testCase.stored),
			}, true)

			middleware := cache.NewMiddleware(
				cache.WithCacheStorage(storage),
				cache.WithMethods([]string{http.MethodGet}),
				cache.WithGlobs(config.CacheGlobs{"/**"}),
			)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/data", nil)
			handler := infra.Wrap(middleware, infra.HandlerFunc(func(contracts.ResponseWriter, *contracts.Request) error {
				t.Fatal("cache hit reached the next handler")

				return nil
			}))
			require.NoError(t, handler.ServeHTTP(server.NewResponseRecorder(recorder), request))

			assert.Equal(t, testCase.want, recorder.Header().Get("Age"))
		})
	}
}
