package infra

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/evg4b/uncors/pkg/urlx"
)

const (
	defaultTimeout  = 5 * time.Minute
	dialerTimeout   = 30 * time.Second
	dialerKeepAlive = 30 * time.Second
)

// createDialContext creates a custom dial context function that uses our DNS resolver.
func createDialContext() func(ctx context.Context, network, addr string) (net.Conn, error) {
	resolver := newDNSResolver()
	dialer := &net.Dialer{
		Timeout:   dialerTimeout,
		KeepAlive: dialerKeepAlive,
	}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
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
}

var defaultHTTPClient = http.Client{
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Transport: &http.Transport{
		Proxy:       http.ProxyFromEnvironment,
		DialContext: createDialContext(),
	},
	Jar:     nil,
	Timeout: defaultTimeout,
}

func MakeHTTPClient(proxy string) *http.Client {
	if len(proxy) > 0 {
		parsedURL, err := urlx.Parse(proxy)
		if err != nil {
			panic(fmt.Errorf("failed to create http client: %w", err))
		}

		httpClient := defaultHTTPClient
		httpClient.Transport = &http.Transport{
			Proxy:       http.ProxyURL(parsedURL),
			DialContext: createDialContext(),
		}

		return &httpClient
	}

	return &defaultHTTPClient
}
