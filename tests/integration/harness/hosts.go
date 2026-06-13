//go:build integration

package harness

import (
	"context"
	"net"
	"regexp"
	"strings"
	"sync"
)

// loopback is where every in-process proxy listener binds.
const loopback = "127.0.0.1"

// Hosts is an in-memory, per-test equivalent of /etc/hosts. It lets a test send
// requests to a real domain (so the Host header and TLS SNI carry that domain,
// which is what uncors routes and mints certificates on) while the underlying TCP
// connection is transparently redirected to the loopback proxy.
//
// Patterns mirror uncors `from` host syntax: exact ("api.example.local"),
// "*.suffix" wildcards, or {key} placeholders — both wildcard forms match a
// single DNS label.
type Hosts struct {
	mu      sync.RWMutex
	entries []hostEntry
}

type hostEntry struct {
	match func(string) bool
	ip    string
}

// NewHosts returns an empty resolver.
func NewHosts() *Hosts {
	return &Hosts{}
}

// Set maps a host pattern to an IP address for the lifetime of the test.
func (h *Hosts) Set(pattern, ip string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries = append(h.entries, hostEntry{match: compileHostPattern(pattern), ip: ip})
}

// DialContext rewrites the connection target for any registered host to its
// mapped IP while preserving the port, so the request keeps its real Host/SNI but
// the connection lands on the in-process proxy. Unregistered hosts dial normally.
func (h *Hosts) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return dialer.DialContext(ctx, network, addr)
	}

	if ip, ok := h.lookup(host); ok {
		addr = net.JoinHostPort(ip, port)
	}

	return dialer.DialContext(ctx, network, addr)
}

// lookup returns the mapped IP for a host, matching the first registered pattern.
func (h *Hosts) lookup(host string) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, entry := range h.entries {
		if entry.match(host) {
			return entry.ip, true
		}
	}

	return "", false
}

// compileHostPattern turns a host pattern into a matcher. {key}/* tokens match a
// single DNS label ([^.]+); everything else is matched literally, case-insensitively.
func compileHostPattern(pattern string) func(string) bool {
	if !strings.ContainsAny(pattern, "{*") {
		want := strings.ToLower(pattern)

		return func(host string) bool { return strings.ToLower(host) == want }
	}

	labels := strings.Split(pattern, ".")
	for i, label := range labels {
		if strings.ContainsAny(label, "{*") {
			labels[i] = "[^.]+"
		} else {
			labels[i] = regexp.QuoteMeta(label)
		}
	}

	re := regexp.MustCompile("(?i)^" + strings.Join(labels, `\.`) + "$")

	return re.MatchString
}
