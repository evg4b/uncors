//go:build integration

package integration

import (
	"context"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/engine"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

type Env struct {
	Client  *http.Client
	Backend *Backend
	Hosts   *Hosts
	FS      afero.Fs

	routes []route
	events *eventRecorder
	proxy  *engine.Engine
}

type route struct {
	pattern string
	match   func(string) bool
	port    int
	scheme  string
}

type Option func(*envOptions)

type envOptions struct {
	files map[string]string
}

func WithFile(path, content string) Option {
	return func(o *envOptions) {
		if o.files == nil {
			o.files = map[string]string{}
		}

		o.files[path] = content
	}
}

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
	env := &Env{Backend: backend, Hosts: resolver, FS: fs}

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

	caCert, recorder, proxy := bootProxy(t, fs, cfg)
	env.Client = newClient(caCert, resolver)
	env.events = recorder
	env.proxy = proxy

	return env
}

func (e *Env) Shutdown(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), proxyShutdownWait)
	defer cancel()

	require.NoError(t, e.proxy.Shutdown(ctx))
}

func (e *Env) Reload(t *testing.T) {
	t.Helper()

	loaded, _, err := config.LoadConfiguration(t.Context(), e.FS, nil, []string{"-c", configPath})
	require.NoError(t, err)
	require.NoError(t, e.proxy.Restart(t.Context(), loaded))
}

func (e *Env) URL(host, path string) string {
	matched := e.routeFor(host)

	scheme := matched.scheme
	if scheme == "" {
		scheme = "https"
	}

	authority := net.JoinHostPort(host, strconv.Itoa(matched.port))

	return testutils.JoinPath(scheme+"://"+authority, path)
}

func (e *Env) RawURL(host, pathAndQuery string) string {
	return strings.TrimSuffix(e.URL(host, "/"), "/") + pathAndQuery
}

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
