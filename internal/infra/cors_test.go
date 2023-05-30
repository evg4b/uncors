package infra_test

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/testing/mocks"
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
					mocks.AllMethods,
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
					mocks.AllMethods,
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
					mocks.AllMethods,
				},
				"X-DATA": []string{"https://demo.com"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			infra.WriteCorsHeaders(tt.header)
		})
	}
}
