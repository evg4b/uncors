package cache_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

func TestCacheableResponseWriter(t *testing.T) {
	const defaultContentType = "text/plain; charset=utf-8"
	const customContentType = "application/xml"
	const authorization = "xxxxxx"
	const bodyString = "Test Body"
	bodyBytes := []byte{0x54, 0x65, 0x73, 0x74, 0x20, 0x42, 0x6f, 0x64, 0x79}

	tests := []struct {
		name     string
		action   func(w http.ResponseWriter)
		expected *cache.CachedResponse
	}{
		{
			name: "write body bytes only",
			action: func(w http.ResponseWriter) {
				sfmt.Fprint(w, bodyString)
			},
			expected: &cache.CachedResponse{
				Header: http.Header{
					headers.ContentType: {defaultContentType},
				},
				Body: bodyBytes,
			},
		},
		{
			name: "write status code only",
			action: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusBadGateway)
			},
			expected: &cache.CachedResponse{
				Header: http.Header{},
				Code:   http.StatusBadGateway,
			},
		},
		{
			name: "write headers only",
			action: func(w http.ResponseWriter) {
				header := w.Header()
				header.Set(headers.ContentType, customContentType)
				header.Set(headers.Authorization, authorization)
			},
			expected: &cache.CachedResponse{
				Header: http.Header{
					headers.ContentType:   {customContentType},
					headers.Authorization: {authorization},
				},
			},
		},
		{
			name: "remove unsaved headers",
			action: func(w http.ResponseWriter) {
				header := w.Header()
				header.Set(headers.ContentLength, "999")
				sfmt.Fprint(w, bodyString)
			},
			expected: &cache.CachedResponse{
				Header: http.Header{
					headers.ContentType: {defaultContentType},
				},
				Body: bodyBytes,
			},
		},
		{
			name: "write full request",
			action: func(writer http.ResponseWriter) {
				header := writer.Header()
				header.Set(headers.ContentType, customContentType)
				header.Set(headers.ContentLength, "9")
				header.Set(headers.Authorization, authorization)
				writer.WriteHeader(http.StatusBadGateway)
				sfmt.Fprint(writer, bodyString)
			},
			expected: &cache.CachedResponse{
				Code: http.StatusBadGateway,
				Header: http.Header{
					headers.ContentType:   {customContentType},
					headers.Authorization: {authorization},
				},
				Body: bodyBytes,
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			cacheableWriter := cache.NewCacheableWriter(recorder)

			testCase.action(cacheableWriter)
			actual := cacheableWriter.GetCachedResponse()

			assert.Equal(t, testCase.expected, actual)
		})
	}
}
