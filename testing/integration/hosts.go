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

func (h *Hosts) Set(pattern, ip string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries = append(h.entries, hostEntry{match: compileHostPattern(pattern), ip: ip})
}

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

		return func(host string) bool { return strings.EqualFold(host, want) }
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
