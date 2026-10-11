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

func corsHeaders(origin, allowHeaders, allowMethods string) http.Header {
	return http.Header{
		headers.AccessControlAllowOrigin:      []string{origin},
		headers.AccessControlAllowCredentials: []string{"true"},
		headers.AccessControlAllowHeaders:     []string{allowHeaders},
		headers.AccessControlAllowMethods:     []string{allowMethods},
		headers.AccessControlMaxAge:           []string{"86400"},
		headers.AccessControlExposeHeaders:    []string{"*"},
	}
}

func TestWriteCorsHeaders(t *testing.T) {
	tests := []struct {
		name     string
		header   http.Header
		origin   string
		expected http.Header
	}{
		{
			name:     "Empty headers without origin",
			header:   http.Header{},
			origin:   "",
			expected: corsHeaders("*", "*", expectedAllowMethods),
		},
		{
			name:     "Empty headers with origin",
			header:   http.Header{},
			origin:   hosts.Localhost.HTTPPort(4000).String(),
			expected: corsHeaders(hosts.Localhost.HTTPPort(4000).String(), "*", expectedAllowMethods),
		},
		{
			name: "Override existing headers",
			header: http.Header{
				headers.AccessControlAllowOrigin:      []string{hosts.Github.HTTPS().String()},
				headers.AccessControlAllowCredentials: []string{"false"},
				headers.AccessControlAllowMethods:     []string{"GET, OPTIONS"},
			},
			origin:   "",
			expected: corsHeaders("*", "*", expectedAllowMethods),
		},
		{
			name: "Do not change existing headers",
			header: http.Header{
				"X-DATA": []string{hosts.Github.HTTPS().String()},
			},
			origin: "",
			expected: http.Header{
				headers.AccessControlAllowOrigin:      []string{"*"},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowHeaders:     []string{"*"},
				headers.AccessControlAllowMethods:     []string{expectedAllowMethods},
				headers.AccessControlMaxAge:           []string{"86400"},
				headers.AccessControlExposeHeaders:    []string{"*"},
				"X-DATA":                              []string{hosts.Github.HTTPS().String()},
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
	testAllowHeaders := "Content-Type, Authorization"
	xCustomHeader := "X-Custom-Header"

	tests := []struct {
		name            string
		reqHeader       http.Header
		expectedHeaders http.Header
	}{
		{
			name:            "Empty request headers",
			reqHeader:       http.Header{},
			expectedHeaders: corsHeaders("*", "*", expectedAllowMethods),
		},
		{
			name: "With Origin header",
			reqHeader: http.Header{
				headers.Origin: []string{hosts.Localhost.HTTPPort(4000).String()},
			},
			expectedHeaders: corsHeaders(hosts.Localhost.HTTPPort(4000).String(), "*", expectedAllowMethods),
		},
		{
			name: "With AccessControlRequestHeaders",
			reqHeader: http.Header{
				headers.AccessControlRequestHeaders: []string{testAllowHeaders},
			},
			expectedHeaders: corsHeaders("*", testAllowHeaders, expectedAllowMethods),
		},
		{
			name: "With AccessControlRequestMethod",
			reqHeader: http.Header{
				headers.AccessControlRequestMethod: []string{"POST"},
			},
			expectedHeaders: corsHeaders("*", "*", "POST"),
		},
		{
			name: "With all request headers",
			reqHeader: http.Header{
				headers.Origin:                      []string{hosts.Github.HTTPS().String()},
				headers.AccessControlRequestHeaders: []string{xCustomHeader},
				headers.AccessControlRequestMethod:  []string{"PUT"},
			},
			expectedHeaders: corsHeaders(hosts.Github.HTTPS().String(), xCustomHeader, "PUT"),
		},
		{
			name: "Overwrite existing response headers",
			reqHeader: http.Header{
				headers.Origin:                     []string{hosts.Example.HTTP().String()},
				headers.AccessControlRequestMethod: []string{"DELETE"},
			},
			expectedHeaders: corsHeaders(hosts.Example.HTTP().String(), "*", "DELETE"),
		},
		{
			name: "Ignore unrelated request headers",
			reqHeader: http.Header{
				headers.Origin:      []string{"http://localhost:3000"},
				xCustomHeader:       []string{"custom-value"},
				headers.UserAgent:   []string{"Mozilla/5.0"},
				headers.ContentType: []string{"application/json"},
			},
			expectedHeaders: corsHeaders("http://localhost:3000", "*", expectedAllowMethods),
		},
		{
			name: "Partial request headers with some defaults",
			reqHeader: http.Header{
				headers.AccessControlRequestHeaders: []string{"Authorization, Content-Type"},
			},
			expectedHeaders: corsHeaders("*", "Authorization, Content-Type", expectedAllowMethods),
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
