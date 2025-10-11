package infra_test

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

func TestWriteCorsHeaders(t *testing.T) {
	tests := []struct {
		name     string
		header   http.Header
		origin   string
		expected http.Header
	}{
		{
			name:   "Empty headers without origin",
			header: http.Header{},
			origin: "",
			expected: http.Header{
				headers.AccessControlAllowOrigin:      []string{"*"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowHeaders:     []string{"*"},
				headers.AccessControlAllowMethods:     []string{"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS"},
				headers.AccessControlMaxAge:           []string{"86400"},
				headers.AccessControlExposeHeaders:    []string{"*"},
			},
		},
		{
			name:   "Empty headers with origin",
			header: http.Header{},
			origin: "http://localhost:4000",
			expected: http.Header{
				headers.AccessControlAllowOrigin:      []string{"http://localhost:4000"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowHeaders:     []string{"*"},
				headers.AccessControlAllowMethods:     []string{"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS"},
				headers.AccessControlMaxAge:           []string{"86400"},
				headers.AccessControlExposeHeaders:    []string{"*"},
			},
		},
		{
			name: "Override existing headers",
			header: http.Header{
				headers.AccessControlAllowOrigin:      []string{hosts.Github.HTTPS()},
				headers.AccessControlAllowCredentials: []string{"false"},
				headers.AccessControlAllowMethods:     []string{"GET, OPTIONS"},
			},
			origin: "",
			expected: http.Header{
				headers.AccessControlAllowOrigin:      []string{"*"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowHeaders:     []string{"*"},
				headers.AccessControlAllowMethods:     []string{"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS"},
				headers.AccessControlMaxAge:           []string{"86400"},
				headers.AccessControlExposeHeaders:    []string{"*"},
			},
		},
		{
			name: "Do not change existing headers",
			header: http.Header{
				"X-DATA": []string{hosts.Github.HTTPS()},
			},
			origin: "",
			expected: http.Header{
				headers.AccessControlAllowOrigin:      []string{"*"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowHeaders:     []string{"*"},
				headers.AccessControlAllowMethods:     []string{"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS"},
				headers.AccessControlMaxAge:           []string{"86400"},
				headers.AccessControlExposeHeaders:    []string{"*"},
				"X-DATA":                              []string{hosts.Github.HTTPS()},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			infra.WriteCorsHeaders(tt.header, tt.origin)

			assert.Equal(t, tt.expected, tt.header)
		})
	}
}

func TestSetHeaderOrDefault(t *testing.T) {
	t.Run("sets value when not empty", func(t *testing.T) {
		header := http.Header{}
		infra.SetHeaderOrDefault(header, "X-Test-Header", "test-value", "default-value")
		assert.Equal(t, "test-value", header.Get("X-Test-Header"))
	})

	t.Run("sets default when value is empty", func(t *testing.T) {
		header := http.Header{}
		infra.SetHeaderOrDefault(header, "X-Test-Header", "", "default-value")
		assert.Equal(t, "default-value", header.Get("X-Test-Header"))
	})

	t.Run("sets default when value is empty string", func(t *testing.T) {
		header := http.Header{}
		infra.SetHeaderOrDefault(header, "X-Test-Header", "", "*")
		assert.Equal(t, "*", header.Get("X-Test-Header"))
	})

	t.Run("overwrites existing header with value", func(t *testing.T) {
		header := http.Header{}
		header.Set("X-Test-Header", "old-value")
		infra.SetHeaderOrDefault(header, "X-Test-Header", "new-value", "default-value")
		assert.Equal(t, "new-value", header.Get("X-Test-Header"))
	})

	t.Run("overwrites existing header with default", func(t *testing.T) {
		header := http.Header{}
		header.Set("X-Test-Header", "old-value")
		infra.SetHeaderOrDefault(header, "X-Test-Header", "", "default-value")
		assert.Equal(t, "default-value", header.Get("X-Test-Header"))
	})
}

func TestWriteCorsHeadersForOptions(t *testing.T) {
	tests := []struct {
		name            string
		reqHeader       http.Header
		expectedHeaders http.Header
	}{
		{
			name:      "Empty request headers",
			reqHeader: http.Header{},
			expectedHeaders: http.Header{
				headers.AccessControlAllowOrigin:      []string{"*"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowHeaders:     []string{"*"},
				headers.AccessControlAllowMethods:     []string{"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS"},
				headers.AccessControlMaxAge:           []string{"86400"},
				headers.AccessControlExposeHeaders:    []string{"*"},
			},
		},
		{
			name: "With Origin header",
			reqHeader: http.Header{
				headers.Origin: []string{"http://localhost:4000"},
			},
			expectedHeaders: http.Header{
				headers.AccessControlAllowOrigin:      []string{"http://localhost:4000"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowHeaders:     []string{"*"},
				headers.AccessControlAllowMethods:     []string{"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS"},
				headers.AccessControlMaxAge:           []string{"86400"},
				headers.AccessControlExposeHeaders:    []string{"*"},
			},
		},
		{
			name: "With AccessControlRequestHeaders",
			reqHeader: http.Header{
				headers.AccessControlRequestHeaders: []string{"Content-Type, Authorization"},
			},
			expectedHeaders: http.Header{
				headers.AccessControlAllowOrigin:      []string{"*"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowHeaders:     []string{"Content-Type, Authorization"},
				headers.AccessControlAllowMethods:     []string{"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS"},
				headers.AccessControlMaxAge:           []string{"86400"},
				headers.AccessControlExposeHeaders:    []string{"*"},
			},
		},
		{
			name: "With AccessControlRequestMethod",
			reqHeader: http.Header{
				headers.AccessControlRequestMethod: []string{"POST"},
			},
			expectedHeaders: http.Header{
				headers.AccessControlAllowOrigin:      []string{"*"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowHeaders:     []string{"*"},
				headers.AccessControlAllowMethods:     []string{"POST"},
				headers.AccessControlMaxAge:           []string{"86400"},
				headers.AccessControlExposeHeaders:    []string{"*"},
			},
		},
		{
			name: "With all request headers",
			reqHeader: http.Header{
				headers.Origin:                      []string{hosts.Github.HTTPS()},
				headers.AccessControlRequestHeaders: []string{"X-Custom-Header"},
				headers.AccessControlRequestMethod:  []string{"PUT"},
			},
			expectedHeaders: http.Header{
				headers.AccessControlAllowOrigin:      []string{hosts.Github.HTTPS()},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowHeaders:     []string{"X-Custom-Header"},
				headers.AccessControlAllowMethods:     []string{"PUT"},
				headers.AccessControlMaxAge:           []string{"86400"},
				headers.AccessControlExposeHeaders:    []string{"*"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respHeader := http.Header{}
			infra.WriteCorsHeadersForOptions(respHeader, tt.reqHeader)

			assert.Equal(t, tt.expectedHeaders, respHeader)
		})
	}
}
