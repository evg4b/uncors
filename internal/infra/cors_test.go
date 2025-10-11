package infra_test

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

const expectedAllowMethods = "GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS"

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
				headers.AccessControlAllowMethods:     []string{expectedAllowMethods},
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
				headers.AccessControlAllowMethods:     []string{expectedAllowMethods},
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
				headers.AccessControlAllowMethods:     []string{expectedAllowMethods},
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
				headers.AccessControlAllowMethods:     []string{expectedAllowMethods},
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
				headers.AccessControlAllowMethods:     []string{expectedAllowMethods},
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
				headers.AccessControlAllowMethods:     []string{expectedAllowMethods},
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
				headers.AccessControlAllowMethods:     []string{expectedAllowMethods},
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
		{
			name: "Overwrite existing response headers",
			reqHeader: http.Header{
				headers.Origin:                     []string{"http://example.com"},
				headers.AccessControlRequestMethod: []string{"DELETE"},
			},
			expectedHeaders: http.Header{
				headers.AccessControlAllowOrigin:      []string{"http://example.com"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowHeaders:     []string{"*"},
				headers.AccessControlAllowMethods:     []string{"DELETE"},
				headers.AccessControlMaxAge:           []string{"86400"},
				headers.AccessControlExposeHeaders:    []string{"*"},
			},
		},
		{
			name: "Ignore unrelated request headers",
			reqHeader: http.Header{
				headers.Origin:      []string{"http://localhost:3000"},
				"X-Custom-Header":   []string{"custom-value"},
				"User-Agent":        []string{"Mozilla/5.0"},
				headers.ContentType: []string{"application/json"},
			},
			expectedHeaders: http.Header{
				headers.AccessControlAllowOrigin:      []string{"http://localhost:3000"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowHeaders:     []string{"*"},
				headers.AccessControlAllowMethods:     []string{expectedAllowMethods},
				headers.AccessControlMaxAge:           []string{"86400"},
				headers.AccessControlExposeHeaders:    []string{"*"},
			},
		},
		{
			name: "Partial request headers with some defaults",
			reqHeader: http.Header{
				headers.AccessControlRequestHeaders: []string{"Authorization, Content-Type"},
			},
			expectedHeaders: http.Header{
				headers.AccessControlAllowOrigin:      []string{"*"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowHeaders:     []string{"Authorization, Content-Type"},
				headers.AccessControlAllowMethods:     []string{expectedAllowMethods},
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
