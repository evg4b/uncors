//go:build integration

package integration

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"sync"
	"testing"

	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/stretchr/testify/assert"
)

// Backend is a recording HTTP server used as the proxy upstream. It stores
// the raw wire-format dump of every request it receives, in arrival order.
type Backend struct {
	server   *httptest.Server
	mu       sync.Mutex
	requests []string
}

// NewBackend starts a plain-HTTP backend and registers shutdown with t.Cleanup.
// When handler is nil every request gets a 200 "ok" text/plain response.
func NewBackend(t *testing.T, handler http.HandlerFunc) *Backend {
	t.Helper()

	backend := &Backend{}

	root := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, err := httputil.DumpRequest(request, true)
		assert.NoError(t, err)

		backend.mu.Lock()
		backend.requests = append(backend.requests, string(raw))
		backend.mu.Unlock()

		if handler != nil {
			handler(writer, request)

			return
		}

		writer.Header().Set("Content-Type", "text/plain")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("ok"))
	})

	backend.server = httptest.NewServer(root)
	t.Cleanup(backend.server.Close)

	return backend
}

// URL returns the backend base URL string (e.g. "http://127.0.0.1:PORT").
func (b *Backend) URL() string {
	return b.server.URL
}

// AsHost returns the backend URL as a urlt.Host for use in config Mapping.To fields.
func (b *Backend) AsHost() urlt.Host {
	parsed, _ := urlt.ParseHost(b.server.URL)

	return *parsed
}

// Count returns the number of requests recorded so far.
func (b *Backend) Count() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return len(b.requests)
}

// Requests returns a copy of all raw HTTP request dumps recorded so far.
func (b *Backend) Requests() []string {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]string, len(b.requests))
	copy(out, b.requests)

	return out
}

// Reset clears all recorded requests.
func (b *Backend) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.requests = nil
}
