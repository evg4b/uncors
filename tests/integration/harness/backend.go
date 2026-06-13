//go:build integration

package harness

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// RecordingBackend is a real HTTP/HTTPS server that records the raw wire-format
// of every request it receives, in arrival order. Recording the raw dump (rather
// than a parsed struct) preserves header ordering, casing, duplicate headers and
// exact body framing, which is what end-to-end snapshots should assert on.
type RecordingBackend struct {
	server   *httptest.Server
	mu       sync.Mutex
	requests []string
}

// NewRecordingBackend starts a backend and registers its shutdown with t.Cleanup.
// When tls is true it serves HTTPS with an httptest-minted certificate. handler
// may be nil, in which case every request gets a 200 "ok".
func NewRecordingBackend(t *testing.T, tls bool, handler http.HandlerFunc) *RecordingBackend {
	t.Helper()

	backend := &RecordingBackend{}

	root := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// DumpRequest reads the body and restores it, so the user handler below
		// still sees an intact request. true => include the body. A failure here
		// can only be a programming error, so surface it without aborting the
		// handler goroutine (require must not be used off the test goroutine).
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

	if tls {
		backend.server = httptest.NewTLSServer(root)
	} else {
		backend.server = httptest.NewServer(root)
	}

	t.Cleanup(backend.server.Close)

	return backend
}

// URL returns the backend's base URL (scheme + host:port).
func (b *RecordingBackend) URL() string {
	return b.server.URL
}

// Count returns the number of requests recorded so far.
func (b *RecordingBackend) Count() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return len(b.requests)
}

// Requests returns a copy of the raw HTTP request dumps recorded so far.
func (b *RecordingBackend) Requests() []string {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]string, len(b.requests))
	copy(out, b.requests)

	return out
}

// Reset clears recorded requests. Call it at the start of each subtest that
// shares one backend so per-subtest count assertions stay independent.
func (b *RecordingBackend) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.requests = nil
}
