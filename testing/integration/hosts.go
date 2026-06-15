//go:build integration

package integration

import (
	"context"
	"net"
	"regexp"
	"strings"
	"sync"
)

const loopback = "127.0.0.1"

// Hosts is an in-memory per-test /etc/hosts equivalent. It lets tests send
// requests to real domain names (so Host and TLS SNI carry that name) while
// the underlying TCP connection is redirected to the loopback proxy.
//
// Patterns follow uncors host syntax: exact ("api.example.local"),
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

func newHosts() *Hosts {
	return &Hosts{}
}

// Set maps a host pattern to an IP address.
func (h *Hosts) Set(pattern, ip string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries = append(h.entries, hostEntry{match: compileHostPattern(pattern), ip: ip})
}

// DialContext rewrites the connection target for registered hosts to their
// mapped IP, preserving the port. Unregistered hosts dial normally.
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
