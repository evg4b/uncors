package infra_test

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testconstants"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
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
					testconstants.AllMethods,
				},
			},
		},
		{
			name: "Override existing headers",
			header: http.Header{
				headers.AccessControlAllowOrigin:      []string{hosts.Github.HTTPS()},
				headers.AccessControlAllowCredentials: []string{"false"},
				headers.AccessControlAllowMethods:     []string{"GET, OPTIONS"},
			},
			expected: http.Header{
				headers.AccessControlAllowOrigin:      []string{"*"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowMethods: []string{
					testconstants.AllMethods,
				},
			},
		},
		{
			name: "Do not change existing headers",
			header: http.Header{
				"X-DATA": []string{hosts.Github.HTTPS()},
			},
			expected: http.Header{
				headers.AccessControlAllowOrigin:      []string{"*"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowMethods: []string{
					testconstants.AllMethods,
				},
				"X-DATA": []string{hosts.Github.HTTPS()},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			infra.WriteCorsHeaders(tt.header)

			assert.Equal(t, tt.expected, tt.header)
		})
	}
}
