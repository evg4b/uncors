package framework

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// EndpointConfig represents a single endpoint configuration for the test server.
type EndpointConfig struct {
	Path     string            `yaml:"path"`
	Method   string            `yaml:"method"`
	Response ResponseConfig    `yaml:"response"`
	Headers  map[string]string `yaml:"headers"`
}

// ResponseConfig defines the response characteristics.
type ResponseConfig struct {
	Status  int               `yaml:"status"`
	Body    string            `yaml:"body"`
	Headers map[string]string `yaml:"headers"`
}

// TestServer wraps httptest.Server with endpoint configuration.
type TestServer struct {
	server    *httptest.Server
	endpoints []EndpointConfig
	mu        sync.RWMutex
	requests  []RecordedRequest
}

// RecordedRequest stores information about received requests.
type RecordedRequest struct {
	Method  string
	Path    string
	Headers http.Header
	Body    string
}

// NewTestServer creates a new test server with the given endpoint configurations.
func NewTestServer(endpoints []EndpointConfig) *TestServer {
	testServer := &TestServer{
		endpoints: endpoints,
		requests:  make([]RecordedRequest, 0),
	}

	testServer.server = httptest.NewServer(http.HandlerFunc(testServer.handler))

	return testServer
}

// NewTestServerTLS creates a new TLS test server with the given endpoint configurations.
func NewTestServerTLS(endpoints []EndpointConfig) *TestServer {
	testServer := &TestServer{
		endpoints: endpoints,
		requests:  make([]RecordedRequest, 0),
	}

	testServer.server = httptest.NewTLSServer(http.HandlerFunc(testServer.handler))

	return testServer
}

//nolint:funcorder,varnamelen // Handler methods grouped with NewTestServer, standard HTTP handler signature
func (ts *TestServer) handler(w http.ResponseWriter, r *http.Request) {
	// Record the request
	bodyBytes, _ := io.ReadAll(r.Body)
	ts.mu.Lock()
	ts.requests = append(ts.requests, RecordedRequest{
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: r.Header.Clone(),
		Body:    string(bodyBytes),
	})
	ts.mu.Unlock()

	// Find matching endpoint
	for _, endpoint := range ts.endpoints {
		if ts.matchEndpoint(endpoint, r.Method, r.URL.Path) {
			ts.serveEndpoint(w, endpoint)

			return
		}
	}

	// No matching endpoint found
	w.WriteHeader(http.StatusNotFound)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"error": "endpoint not found",
	}); err != nil {
		// Ignore error - response already sent
		_ = err
	}
}

//nolint:funcorder // Handler methods grouped with NewTestServer
func (ts *TestServer) matchEndpoint(endpoint EndpointConfig, method, path string) bool {
	methodMatch := strings.EqualFold(endpoint.Method, method)
	pathMatch := endpoint.Path == path

	return methodMatch && pathMatch
}

//nolint:funcorder,varnamelen // Handler methods grouped with NewTestServer, standard HTTP handler signature
func (ts *TestServer) serveEndpoint(w http.ResponseWriter, endpoint EndpointConfig) {
	// Set response headers
	for key, value := range endpoint.Response.Headers {
		w.Header().Set(key, value)
	}

	// Set status code
	status := endpoint.Response.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)

	// Write body
	if endpoint.Response.Body != "" {
		if _, err := w.Write([]byte(endpoint.Response.Body)); err != nil {
			// Ignore error - response already started
			_ = err
		}
	}
}

// URL returns the base URL of the test server.
func (ts *TestServer) URL() string {
	return ts.server.URL
}

// Close shuts down the test server.
func (ts *TestServer) Close() {
	ts.server.Close()
}

// GetRequests returns all recorded requests.
func (ts *TestServer) GetRequests() []RecordedRequest {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	return append([]RecordedRequest{}, ts.requests...)
}

// ClearRequests clears all recorded requests.
func (ts *TestServer) ClearRequests() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.requests = make([]RecordedRequest, 0)
}

// GetPort extracts the port number from the server URL.
func (ts *TestServer) GetPort() string {
	url := ts.server.URL
	parts := strings.Split(url, ":")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}

// String returns a string representation of the test server.
func (ts *TestServer) String() string {
	return fmt.Sprintf("TestServer{URL: %s, Endpoints: %d}", ts.URL(), len(ts.endpoints))
}
