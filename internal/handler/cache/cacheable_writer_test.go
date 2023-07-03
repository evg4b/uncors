package cache_test

import (
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCacheableResponseWriter(t *testing.T) {
	const defaultContentType = "text/plain; charset=utf-8"
	const customContentType = "application/xml"
	const authorization = "xxxxxx"
	const bodyString = "Test Body"
	var bodyBytes = []byte{0x54, 0x65, 0x73, 0x74, 0x20, 0x42, 0x6f, 0x64, 0x79}

	tests := []struct {
		name     string
		action   func(w http.ResponseWriter)
		expected *cache.CachedResponse
	}{
		{
			name: "Write bodyBytes only",
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
			name: "Write status code only",
			action: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusBadGateway)
			},
			expected: &cache.CachedResponse{
				Header: http.Header{},
				Code:   http.StatusBadGateway,
			},
		},
		{
			name: "Write headers only",
			action: func(w http.ResponseWriter) {
				header := w.Header()
				header.Set(headers.ContentType, customContentType)
				header.Set(headers.ContentLength, "0")
				header.Set(headers.Authorization, authorization)
			},
			expected: &cache.CachedResponse{
				Header: http.Header{
					headers.ContentType:   {customContentType},
					headers.ContentLength: {"0"},
					headers.Authorization: {authorization},
				},
			},
		},
		{
			name: "Write full request",
			action: func(w http.ResponseWriter) {
				header := w.Header()
				header.Set(headers.ContentType, customContentType)
				header.Set(headers.ContentLength, "9")
				header.Set(headers.Authorization, authorization)
				w.WriteHeader(http.StatusBadGateway)
				sfmt.Fprint(w, bodyString)
			},
			expected: &cache.CachedResponse{
				Code: http.StatusBadGateway,
				Header: http.Header{
					headers.ContentType:   {customContentType},
					headers.ContentLength: {"9"},
					headers.Authorization: {authorization},
				},
				Body: bodyBytes,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			cacheableWriter := cache.NewCacheableWriter(recorder)

			tt.action(cacheableWriter)
			actual := cacheableWriter.GetCachedResponse()

			assert.Equal(t, tt.expected, actual)
		})
	}
}
