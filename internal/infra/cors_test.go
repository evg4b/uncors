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
				headers.AccessControlAllowMethods:     []string{"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, LINK, OPTIONS"},
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
				headers.AccessControlAllowMethods:     []string{"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, LINK, OPTIONS"},
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
				headers.AccessControlAllowMethods:     []string{"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, LINK, OPTIONS"},
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
				headers.AccessControlAllowMethods:     []string{"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, LINK, OPTIONS"},
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
