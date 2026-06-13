//go:build integration

package harness

import (
	"net"
	"strconv"
	"strings"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/testutils"
)

// RouteSpec declares one domain-based proxy mapping for a test: a local domain
// (optionally with a {key}/* placeholder) forwarded to an upstream.
type RouteSpec struct {
	host     string // host pattern without scheme or port, e.g. "{sub}.example.local"
	tls      bool
	to       string // upstream; empty => the env backend
	decorate func(*config.Mapping)

	port int // assigned by the harness
}

// RouteOption customises a RouteSpec.
type RouteOption func(*RouteSpec)

// RouteOverHTTP serves the route over plain HTTP instead of the default HTTPS.
func RouteOverHTTP() RouteOption {
	return func(r *RouteSpec) { r.tls = false }
}

// RouteTo overrides the upstream target (defaults to the env backend).
func RouteTo(upstream string) RouteOption {
	return func(r *RouteSpec) { r.to = upstream }
}

// RouteWithMapping decorates the generated mapping, e.g. to attach mocks or cache.
func RouteWithMapping(decorate func(*config.Mapping)) RouteOption {
	return func(r *RouteSpec) { r.decorate = decorate }
}

// Route is a handle to a started domain mapping, used to build request URLs.
type Route struct {
	pattern string
	tls     bool
	port    int
}

// Port returns the ephemeral listener port the route is served on.
func (r *Route) Port() int {
	return r.port
}

// Host resolves the route's host pattern for a concrete subdomain label. For a
// placeholder/wildcard pattern the first label is substituted; for an exact
// pattern the label is ignored.
func (r *Route) Host(label string) string {
	host := r.pattern
	if strings.ContainsAny(host, "{*") {
		host = label + host[strings.IndexByte(host, '.'):]
	}

	return host
}

// URL builds an absolute request URL for the route at the given subdomain label
// and path. Pass an empty label for exact (non-placeholder) routes.
func (r *Route) URL(label, path string) string {
	scheme := "https"
	if !r.tls {
		scheme = "http"
	}

	authority := net.JoinHostPort(r.Host(label), strconv.Itoa(r.port))

	return testutils.JoinPath(scheme+"://"+authority, path)
}
