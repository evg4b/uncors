//go:build integration

package integration

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// Env is the wired test environment: a recording backend, an in-process proxy
// and a client that trusts the proxy's CA and resolves mapped hosts in-memory.
//
// Every mapping is reached through its real From hostname (never a raw loopback
// IP): the harness assigns each mapping a free port, registers the hostname in
// the in-memory resolver, and the client dials the host transparently to the
// loopback proxy with the Host header and TLS SNI intact.
type Env struct {
	// Client is an HTTP client pre-configured to trust the proxy's dev CA and
	// to resolve every mapped host to the in-process proxy.
	Client *http.Client
	// Backend is the recording upstream server.
	Backend *Backend
	// Hosts is the in-memory resolver mapping every From host to the loopback
	// proxy. Use Hosts.DialContext when building a custom client (e.g. one that
	// does not trust the proxy CA) so it can still reach mapped hosts by name.
	Hosts *Hosts

	routes []route
}

type route struct {
	pattern string
	match   func(string) bool
	port    int
	scheme  string
}

// Option customises the Env.
type Option func(*envOptions)

type envOptions struct {
	files map[string]string
}

// WithFile seeds the proxy filesystem with a file at path, used for static
// file serving, file-backed mocks, or Lua scripts loaded from disk.
func WithFile(path, content string) Option {
	return func(o *envOptions) {
		if o.files == nil {
			o.files = map[string]string{}
		}

		o.files[path] = content
	}
}

// New wires backend → proxy → client and registers all teardown with t.Cleanup.
//
// cfg is the complete proxy configuration. Every Mapping.From must be a named
// host (exact like "api.example.local" or a {placeholder} pattern) without a
// port: the harness assigns a free port and registers the host so the client
// can reach it via Env.URL. Mapping.To is normally backend.AsHost().
func New(t *testing.T, backend *Backend, cfg *config.UncorsConfig, opts ...Option) *Env {
	t.Helper()

	options := &envOptions{}
	for _, opt := range opts {
		opt(options)
	}

	fs := afero.NewMemMapFs()
	for path, content := range options.files {
		require.NoError(t, afero.WriteFile(fs, path, []byte(content), os.ModePerm))
	}

	resolver := newHosts()
	env := &Env{Backend: backend, Hosts: resolver}

	for i := range cfg.Mappings {
		from := &cfg.Mappings[i].From
		require.NotEmpty(t, from.Hostname, "mapping From host must be a named host, not empty")

		// A From without a port gets a fresh one; an explicit port is respected
		// so several mappings can deliberately share one listener (host routing).
		if from.Port == "" {
			from.Port = strconv.Itoa(testutils.GetFreePort(t))
		}

		port, err := strconv.Atoi(from.Port)
		require.NoError(t, err)

		resolver.Set(from.Hostname, loopback)
		env.routes = append(env.routes, route{
			pattern: from.Hostname,
			match:   compileHostPattern(from.Hostname),
			port:    port,
			scheme:  from.Scheme,
		})
	}

	caCert := bootProxy(t, fs, cfg)
	env.Client = newClient(caCert, resolver)

	return env
}

// URL builds an absolute request URL for the mapping whose From host matches
// the given concrete hostname. For a {placeholder} mapping pass the resolved
// subdomain (e.g. "acme.api.local"); for an exact mapping pass that host.
//
// Note: the query string is not appended here (the path joiner would encode
// "?"). Append "?key=value" to the returned URL when a query is needed.
func (e *Env) URL(host, path string) string {
	matched := e.routeFor(host)

	scheme := matched.scheme
	if scheme == "" {
		scheme = "https"
	}

	authority := net.JoinHostPort(host, strconv.Itoa(matched.port))

	return testutils.JoinPath(scheme+"://"+authority, path)
}

// PortFor returns the listener port assigned to the mapping whose From host
// matches the given concrete hostname.
func (e *Env) PortFor(host string) int {
	return e.routeFor(host).port
}

func (e *Env) routeFor(host string) route {
	for _, r := range e.routes {
		if r.match(host) {
			return r
		}
	}

	panic("integration: no mapping registered for host " + host)
}
