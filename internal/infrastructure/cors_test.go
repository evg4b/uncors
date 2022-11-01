package infrastructure_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

func TestWriteCorsHeaders(t *testing.T) {
	tests := []struct {
		name            string
		recorderFactory func() *httptest.ResponseRecorder
		expected        http.Header
	}{
		{
			name:            "should append data in empty writer",
			recorderFactory: httptest.NewRecorder,
			expected: map[string][]string{
				headers.AccessControlAllowOrigin:      {"*"},
				headers.AccessControlAllowCredentials: {"true"},
				headers.AccessControlAllowMethods: {
					"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
				},
			},
		},
		{
			name: "should append data in filled writer",
			recorderFactory: func() *httptest.ResponseRecorder {
				writer := httptest.NewRecorder()
				writer.Header().Set("Test-Header", "true")
				writer.Header().Set("X-Hey-Header", "123")

				return writer
			},
			expected: map[string][]string{
				"Test-Header":                         {"true"},
				"X-Hey-Header":                        {"123"},
				headers.AccessControlAllowOrigin:      {"*"},
				headers.AccessControlAllowCredentials: {"true"},
				headers.AccessControlAllowMethods: {
					"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
				},
			},
		},
		{
			name: "should override same headers",
			recorderFactory: func() *httptest.ResponseRecorder {
				writer := httptest.NewRecorder()
				writer.Header().Set("Test-Header", "true")
				writer.Header().Set(headers.AccessControlAllowOrigin, "localhost:3000")

				return writer
			},
			expected: map[string][]string{
				"Test-Header":                         {"true"},
				headers.AccessControlAllowOrigin:      {"*"},
				headers.AccessControlAllowCredentials: {"true"},
				headers.AccessControlAllowMethods: {
					"GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS",
				},
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			resp := testCase.recorderFactory()
			infrastructure.WriteCorsHeaders(resp.Header())

			response := resp.Result()
			defer testutils.CheckNoError(t, response.Body.Close())

			assert.Equal(t, response.Header, testCase.expected)
		})
	}
}
