package framework

import (
	"context"
	"net"
	"net/http"
	"time"
)

const (
	testDialerTimeout   = 30 * time.Second
	testDialerKeepAlive = 30 * time.Second
)

// createTestHTTPClient creates an HTTP client with custom DNS resolver for integration tests.
func createTestHTTPClient() *http.Client {
	resolver := newTestDNSResolver()
	dialer := &net.Dialer{
		Timeout:   testDialerTimeout,
		KeepAlive: testDialerKeepAlive,
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		// Use custom DNS resolver
		ips, err := resolver.LookupHost(ctx, host)
		if err != nil {
			return nil, err
		}

		// Use the first resolved IP
		if len(ips) > 0 {
			addr = net.JoinHostPort(ips[0], port)
		}

		return dialer.DialContext(ctx, network, addr)
	}

	return &http.Client{
		Timeout: requestTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			DialContext: dialContext,
		},
	}
}

// testDNSResolver is the interface for DNS resolution in tests.
type testDNSResolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
}
