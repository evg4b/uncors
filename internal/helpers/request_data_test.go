package helpers_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
)

type mockResponseWriter struct {
	statusCode int
}

func (m *mockResponseWriter) Header() http.Header {
	return make(http.Header)
}

func (m *mockResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

func (m *mockResponseWriter) StatusCode() int {
	return m.statusCode
}

func headersEqual(header1, header2 http.Header) bool {
	if len(header1) != len(header2) {
		return false
	}

	for key, values := range header1 {
		if len(values) != len(header2[key]) {
			return false
		}

		for i, v := range values {
			if v != header2[key][i] {
				return false
			}
		}
	}

	return true
}

func TestToRequestData(t *testing.T) {
	testCases := []struct {
		name           string
		method         string
		urlStr         string
		headers        http.Header
		statusCode     int
		expectedMethod string
		expectedCode   int
	}{
		{
			name:           "GET request with 200 status",
			method:         http.MethodGet,
			urlStr:         "http://example.com/path",
			headers:        http.Header{"X-Custom": []string{"value"}},
			statusCode:     200,
			expectedMethod: http.MethodGet,
			expectedCode:   200,
		},
		{
			name:           "POST request with 201 status",
			method:         http.MethodPost,
			urlStr:         "https://api.example.com/users",
			headers:        http.Header{"Content-Type": []string{"application/json"}},
			statusCode:     201,
			expectedMethod: http.MethodPost,
			expectedCode:   201,
		},
		{
			name:           "DELETE request with 404 status",
			method:         http.MethodDelete,
			urlStr:         "http://example.com/resource/123",
			headers:        http.Header{},
			statusCode:     404,
			expectedMethod: http.MethodDelete,
			expectedCode:   404,
		},
		{
			name:           "PUT request with 500 status",
			method:         http.MethodPut,
			urlStr:         "http://example.com/",
			headers:        http.Header{"Authorization": []string{"Bearer token"}},
			statusCode:     500,
			expectedMethod: http.MethodPut,
			expectedCode:   500,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			url, _ := url.Parse(testCase.urlStr)
			req := &contracts.Request{
				Method: testCase.method,
				URL:    url,
				Header: testCase.headers,
			}

			res := &mockResponseWriter{statusCode: testCase.statusCode}

			result := helpers.ToRequestData(req, res)

			if result.Method != testCase.expectedMethod {
				t.Errorf("expected method %s, got %s", testCase.expectedMethod, result.Method)
			}

			if result.Code != testCase.expectedCode {
				t.Errorf("expected status code %d, got %d", testCase.expectedCode, result.Code)
			}

			if result.URL != url {
				t.Errorf("expected URL %v, got %v", url, result.URL)
			}

			if !headersEqual(result.Header, testCase.headers) {
				t.Errorf("expected headers %v, got %v", testCase.headers, result.Header)
			}

			if result.Body != nil {
				t.Errorf("expected body to be nil, got %v", result.Body)
			}
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

	res := &mockResponseWriter{statusCode: 200}

	result := helpers.ToRequestData(req, res)

	if result.URL.String() != url.String() {
		t.Errorf("expected URL %s, got %s", url.String(), result.URL.String())
	}
}
