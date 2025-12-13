package cache_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheableResponseWriter(t *testing.T) {
	type testCase struct {
		name     string
		action   func(w http.ResponseWriter)
		expected *contracts.CachedResponse
	}

	pngSignature := []byte{
		0x89, 0x50, 0x4E, 0x47,
		0x0D, 0x0A, 0x1A, 0x0A,
	}

	tests := []testCase{
		{
			name:   "empty handler",
			action: func(_ http.ResponseWriter) {},
			expected: &contracts.CachedResponse{
				Body:    nil,
				Code:    http.StatusOK,
				Headers: []contracts.CachedHeader{},
			},
		},
		{
			name: "write body only",
			action: func(writer http.ResponseWriter) {
				fmt.Fprint(writer, "test")
			},
			expected: &contracts.CachedResponse{
				Body: []byte("test"),
				Code: http.StatusOK,
				Headers: []contracts.CachedHeader{
					testutils.CachedHeader(headers.ContentType, "text/plain; charset=utf-8"),
				},
			},
		},
		{
			name: "status 200 only",
			action: func(writer http.ResponseWriter) {
				writer.WriteHeader(http.StatusOK)
			},
			expected: &contracts.CachedResponse{
				Body:    nil,
				Code:    http.StatusOK,
				Headers: []contracts.CachedHeader{},
			},
		},
		{
			name: "headers only",
			action: func(writer http.ResponseWriter) {
				header := writer.Header()
				header.Set(headers.XForwardedFor, "127.0.0.1")
				header.Set(headers.XForwardedProto, "https")
				header.Set(headers.XPoweredBy, "uncors")
			},
			expected: &contracts.CachedResponse{
				Body: nil,
				Code: http.StatusOK,
				Headers: []contracts.CachedHeader{
					testutils.CachedHeader(headers.XForwardedFor, "127.0.0.1"),
					testutils.CachedHeader(headers.XForwardedProto, "https"),
					testutils.CachedHeader(headers.XPoweredBy, "uncors"),
				},
			},
		},
		{
			name: "full filled response",
			action: func(writer http.ResponseWriter) {
				writer.WriteHeader(http.StatusCreated)

				header := writer.Header()
				header.Add(headers.ContentType, "image/png")
				header.Add(headers.CacheControl, "no-cache")
				header.Add(headers.XPoweredBy, "uncors")

				_, err := writer.Write(pngSignature)
				require.NoError(t, err)
			},
			expected: &contracts.CachedResponse{
				Code: http.StatusCreated,
				Body: pngSignature,
				Headers: []contracts.CachedHeader{
					testutils.CachedHeader(headers.ContentType, "image/png"),
					testutils.CachedHeader(headers.CacheControl, "no-cache"),
					testutils.CachedHeader(headers.XPoweredBy, "uncors"),
				},
			},
		},
	}

	statusCodes := []int{
		http.StatusContinue,
		http.StatusSwitchingProtocols,
		http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusGatewayTimeout,
	}
	statusCodeTestCases := lo.Map(statusCodes, func(code int, _ int) testCase {
		return testCase{
			name: fmt.Sprintf("for status code %d ignore caching", code),
			action: func(writer http.ResponseWriter) {
				writer.WriteHeader(code)
				writer.Header().Add(headers.ContentType, "text/plain; charset=utf-8")
				fmt.Fprintf(writer, "status code: %d", code)
			},
		}
	})

	for _, testCase := range slices.Concat(tests, statusCodeTestCases) {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			cacheMock := mocks.NewCacheMock(t)
			cacheKey := lo.RandomString(5, lo.LettersCharset)

			if testCase.expected != nil {
				cacheMock.SetMock.Expect(cacheKey, *testCase.expected)
			}

			cacheableWriter := cache.NewCacheableResponseWriter(cacheMock, recorder, cacheKey)

			testCase.action(cacheableWriter)
			cacheableWriter.Close()
		})
	}

	for _, code := range slices.Concat(statusCodes, []int{http.StatusOK, http.StatusNoContent}) {
		t.Run(fmt.Sprintf("writer should return %d code", code), func(t *testing.T) {
			recorder := httptest.NewRecorder()

			cacheMock := mocks.NewCacheMock(t)
			if helpers.Is2xxCode(code) {
				cacheMock.SetMock.Set(func(_ string, _ contracts.CachedResponse) {})
			}

			cacheableWriter := cache.NewCacheableResponseWriter(cacheMock, recorder, "cache-key")

			cacheableWriter.WriteHeader(code)
			defer cacheableWriter.Close()

			assert.Equal(t, code, cacheableWriter.StatusCode())
		})
	}
}
