package helpers_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToRequestData(t *testing.T) {
	testCases := []struct {
		name       string
		method     string
		urlStr     string
		headers    http.Header
		statusCode int
	}{
		{
			name:       "GET request with 200 status",
			method:     http.MethodGet,
			urlStr:     "http://example.com/path",
			headers:    http.Header{"X-Custom": []string{"value"}},
			statusCode: 200,
		},
		{
			name:       "POST request with 201 status",
			method:     http.MethodPost,
			urlStr:     "https://api.example.com/users",
			headers:    http.Header{"Content-Type": []string{"application/json"}},
			statusCode: 201,
		},
		{
			name:       "DELETE request with 404 status",
			method:     http.MethodDelete,
			urlStr:     "http://example.com/resource/123",
			headers:    http.Header{},
			statusCode: 404,
		},
		{
			name:       "PUT request with 500 status",
			method:     http.MethodPut,
			urlStr:     "http://example.com/",
			headers:    http.Header{"Authorization": []string{"Bearer token"}},
			statusCode: 500,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parsedURL, err := url.Parse(testCase.urlStr)
			require.NoError(t, err)

			req := &contracts.Request{
				Method: testCase.method,
				URL:    parsedURL,
				Header: testCase.headers,
			}

			result := helpers.ToRequestData(req, testCase.statusCode)

			assert.Equal(t, testCase.method, result.Method)
			assert.Equal(t, testCase.statusCode, result.Code)
			assert.Equal(t, parsedURL, result.URL)
			assert.Equal(t, testCase.headers, result.Header)
			assert.Nil(t, result.Body)
		})
	}
}

func TestToRequestDataPreservesURL(t *testing.T) {
	url, _ := url.Parse("http://example.com/path?query=value#fragment")
	req := &contracts.Request{
		Method: http.MethodGet,
		URL:    url,
		Header: make(http.Header),
	}

	result := helpers.ToRequestData(req, 200)

	if result.URL.String() != url.String() {
		t.Errorf("expected URL %s, got %s", url.String(), result.URL.String())
	}
}
