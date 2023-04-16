package infrastructure_test

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/go-http-utils/headers"
)

func TestWriteCorsHeaders(t *testing.T) {
	tests := []struct {
		name     string
		header   http.Header
		expected http.Header
	}{
		{
			name:   "Empty headers",
			header: http.Header{},
			expected: http.Header{
				headers.AccessControlAllowOrigin:      []string{"*"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowMethods: []string{
					"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
				},
			},
		},
		{
			name: "Override existing headers",
			header: http.Header{
				headers.AccessControlAllowOrigin:      []string{"https://demo.com"},
				headers.AccessControlAllowCredentials: []string{"false"},
				headers.AccessControlAllowMethods:     []string{"GET, OPTIONS"},
			},
			expected: http.Header{
				headers.AccessControlAllowOrigin:      []string{"*"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowMethods: []string{
					"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
				},
			},
		},
		{
			name: "Do not change existing headers",
			header: http.Header{
				"X-DATA": []string{"https://demo.com"},
			},
			expected: http.Header{
				headers.AccessControlAllowOrigin:      []string{"*"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowMethods: []string{
					"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
				},
				"X-DATA": []string{"https://demo.com"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			infrastructure.WriteCorsHeaders(tt.header)
		})
	}
}
