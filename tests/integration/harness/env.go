//go:build integration

package harness

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/config"
)

// Env is a fully wired scenario: a recording backend, an in-process proxy
// forwarding to it, and a client that trusts the proxy's CA. All teardown is
// registered via t.Cleanup, so there is nothing to defer at the call site.
type Env struct {
	Backend *RecordingBackend
	Proxy   *ProxyHarness
	Client  *http.Client
}

type options struct {
	backendTLS bool
	handler    http.HandlerFunc
	decorate   func(*config.Mapping)
}

// Option customises an Env built by New.
type Option func(*options)

// WithBackendHandler sets the backend's request handler. Without it the backend
// answers every request with 200 "ok".
func WithBackendHandler(handler http.HandlerFunc) Option {
	return func(o *options) { o.handler = handler }
}

// WithMapping decorates the proxy's forwarding mappings, e.g. to attach mocks,
// rewrites or caching.
func WithMapping(decorate func(*config.Mapping)) Option {
	return func(o *options) { o.decorate = decorate }
}

// New wires backend -> proxy -> client and registers cleanup. The backend serves
// plain HTTP by design: the proxy's upstream client verifies against the system
// root pool and has no injection seam, so the meaningful TLS surface is the
// client<->proxy hop, which uses the proxy's trusted dev CA.
func New(t *testing.T, opts ...Option) *Env {
	t.Helper()

	cfg := &options{}
	for _, opt := range opts {
		opt(cfg)
	}

	backend := NewRecordingBackend(t, cfg.backendTLS, cfg.handler)
	proxy := NewProxyHarness(t, backend.URL(), cfg.decorate)

	return &Env{
		Backend: backend,
		Proxy:   proxy,
		Client:  NewClient(proxy.CACert()),
	}
}
