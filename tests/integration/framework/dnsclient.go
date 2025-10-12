package framework

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// DNSResolver provides custom DNS resolution for testing.
type DNSResolver struct {
	mappings map[string]string
}

// NewDNSResolver creates a new DNS resolver with the given hostname mappings.
func NewDNSResolver(mappings map[string]string) *DNSResolver {
	if mappings == nil {
		mappings = make(map[string]string)
	}
	return &DNSResolver{
		mappings: mappings,
	}
}

// Resolve returns the IP address for the given hostname.
func (r *DNSResolver) Resolve(hostname string) (string, bool) {
	ip, ok := r.mappings[hostname]
	return ip, ok
}

// AddMapping adds a hostname to IP address mapping.
func (r *DNSResolver) AddMapping(hostname, ip string) {
	r.mappings[hostname] = ip
}

// CreateHTTPClient creates an HTTP client with custom DNS resolution.
func CreateHTTPClient(resolver *DNSResolver) *http.Client {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			// Check if we have a custom DNS mapping
			if ip, ok := resolver.Resolve(host); ok {
				addr = net.JoinHostPort(ip, port)
			}

			return dialer.DialContext(ctx, network, addr)
		},
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects - return the redirect response
			return http.ErrUseLastResponse
		},
	}
}

// CreateHTTPClientWithDefaults creates an HTTP client with default localhost mappings.
func CreateHTTPClientWithDefaults() *http.Client {
	resolver := NewDNSResolver(map[string]string{
		"localhost": "127.0.0.1",
	})
	return CreateHTTPClient(resolver)
}

// ResolveHostPort resolves a host:port combination using the DNS resolver.
func (r *DNSResolver) ResolveHostPort(hostPort string) (string, error) {
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return "", fmt.Errorf("failed to split host and port: %w", err)
	}

	if ip, ok := r.Resolve(host); ok {
		return net.JoinHostPort(ip, port), nil
	}

	return hostPort, nil
}
