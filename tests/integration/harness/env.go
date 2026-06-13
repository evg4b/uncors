//go:build integration

package harness

import (
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// Env is a fully wired scenario: a recording backend, an in-process proxy
// forwarding to it, a per-test in-memory host resolver, and a client that trusts
// the proxy's CA and honours that resolver. All teardown is registered via
// t.Cleanup, so there is nothing to defer at the call site.
type Env struct {
	Backend *RecordingBackend
	Proxy   *ProxyHarness
	Client  *http.Client
	Hosts   *Hosts
	FS      afero.Fs // the proxy's filesystem; seed it for static/mock-file/script

	routes map[string]*Route
}

type options struct {
	backendTLS  bool
	handler     http.HandlerFunc
	decorate    func(*config.Mapping)
	routes      []*RouteSpec
	cacheConfig config.CacheConfig
	files       map[string]string
}

// Option customises an Env built by New.
type Option func(*options)

// WithBackendHandler sets the backend's request handler. Without it the backend
// answers every request with 200 "ok".
func WithBackendHandler(handler http.HandlerFunc) Option {
	return func(o *options) { o.handler = handler }
}

// WithMapping decorates the proxy's default loopback mappings, e.g. to attach
// mocks, rewrites or caching.
func WithMapping(decorate func(*config.Mapping)) Option {
	return func(o *options) { o.decorate = decorate }
}

// WithCacheConfig sets the proxy-wide cache configuration (methods, size, TTL).
// Per-route cache globs are attached via WithMapping (mapping.Cache).
func WithCacheConfig(cacheConfig config.CacheConfig) Option {
	return func(o *options) { o.cacheConfig = cacheConfig }
}

// WithFile seeds the proxy filesystem (Env.FS) with a file, for static serving,
// file-backed mocks or Lua scripts loaded from disk.
func WithFile(path, content string) Option {
	return func(o *options) {
		if o.files == nil {
			o.files = map[string]string{}
		}

		o.files[path] = content
	}
}

// WithDomain adds a domain-based mapping: the local domain host (optionally with
// a {key}/* placeholder) is served on its own listener and forwarded to the
// upstream (the backend by default). The domain is registered in the Env's host
// resolver so the client can reach it. Retrieve a handle with Env.Route.
func WithDomain(host string, opts ...RouteOption) Option {
	return func(cfg *options) {
		spec := &RouteSpec{host: host, tls: true}
		for _, opt := range opts {
			opt(spec)
		}

		cfg.routes = append(cfg.routes, spec)
	}
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
	resolver := NewHosts()

	fs := afero.NewMemMapFs()
	for path, content := range cfg.files {
		require.NoError(t, afero.WriteFile(fs, path, []byte(content), os.ModePerm))
	}

	httpPort := testutils.GetFreePort(t)
	httpsPort := testutils.GetFreePort(t)

	const loopbackRoutes = 2

	mappings := make(config.Mappings, 0, loopbackRoutes+len(cfg.routes))
	mappings = append(mappings,
		decorated(config.Mapping{From: hosts.Loopback.HTTPPort(httpPort), To: backend.URL()}, cfg.decorate),
		decorated(config.Mapping{From: hosts.Loopback.HTTPSPort(httpsPort), To: backend.URL()}, cfg.decorate),
	)

	routes := make(map[string]*Route, len(cfg.routes))
	for _, spec := range cfg.routes {
		spec.port = testutils.GetFreePort(t)

		upstream := spec.to
		if upstream == "" {
			upstream = backend.URL()
		}

		from := spec.host + ":" + strconv.Itoa(spec.port)
		mapping := config.Mapping{From: scheme(spec.tls) + "://" + from, To: upstream}
		mappings = append(mappings, decorated(mapping, spec.decorate))

		// In-memory /etc/hosts: the real domain resolves to the loopback proxy.
		resolver.Set(spec.host, loopback)
		routes[spec.host] = &Route{pattern: spec.host, tls: spec.tls, port: spec.port}
	}

	caCert := bootProxy(t, fs, &config.UncorsConfig{Mappings: mappings, CacheConfig: cfg.cacheConfig})

	return &Env{
		Backend: backend,
		Proxy:   &ProxyHarness{caCert: caCert, HTTPPort: httpPort, HTTPSPort: httpsPort},
		Client:  NewClient(caCert, resolver),
		Hosts:   resolver,
		FS:      fs,
		routes:  routes,
	}
}

// Route returns the handle for a domain registered via WithDomain. It fails the
// test if the host was not registered.
func (e *Env) Route(t *testing.T, host string) *Route {
	t.Helper()

	route, ok := e.routes[host]
	if !ok {
		t.Fatalf("no route registered for host %q", host)
	}

	return route
}

func decorated(mapping config.Mapping, decorate func(*config.Mapping)) config.Mapping {
	if decorate != nil {
		decorate(&mapping)
	}

	return mapping
}

func scheme(tls bool) string {
	if tls {
		return "https"
	}

	return "http"
}
