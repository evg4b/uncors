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

type Backend struct {
	server   *httptest.Server
	mu       sync.Mutex
	requests []string
}

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

func (b *Backend) URL() string {
	return b.server.URL
}

func (b *Backend) AsHost() urlt.Host {
	parsed, _ := urlt.ParseHost(b.server.URL)

	return *parsed
}

func (b *Backend) Count() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return len(b.requests)
}

func (b *Backend) Requests() []string {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]string, len(b.requests))
	copy(out, b.requests)

	return out
}

func (b *Backend) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.requests = nil
}
