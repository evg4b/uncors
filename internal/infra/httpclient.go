package infra

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	// maxIdleConnsPerHost is well above the ~6 parallel connections a browser
	// opens per host, so a browser driven workload reuses connections instead of
	// paying for a TCP and TLS handshake per request. The standard library's
	// default of 2 makes that impossible.
	maxIdleConnsPerHost = 32

	// responseHeaderTimeout bounds an upstream that accepts the connection and
	// then never replies. There is deliberately no whole-request timeout: that
	// would cap long downloads and make streamed responses impossible. A client
	// that gives up cancels the inbound request context instead.
	responseHeaderTimeout = 60 * time.Second
)

// errUnexpectedDefaultTransport guards the assumption that the standard
// library's default transport is an *http.Transport we can clone.
var errUnexpectedDefaultTransport = errors.New("http.DefaultTransport is not an *http.Transport")

// ClientPool hands out the HTTP clients used to talk to upstreams. A transport
// owns a connection pool and is meant to be shared and long lived, so there is
// one client per distinct upstream proxy setting rather than one per handler.
type ClientPool struct {
	mu      sync.Mutex
	clients map[string]*http.Client
}

func NewClientPool() *ClientPool {
	return &ClientPool{clients: map[string]*http.Client{}}
}

// For returns the client for the given upstream proxy ("" means "use the
// environment"), creating it on first use.
func (p *ClientPool) For(proxy string) (*http.Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if client, ok := p.clients[proxy]; ok {
		return client, nil
	}

	client, err := newHTTPClient(proxy)
	if err != nil {
		return nil, err
	}

	p.clients[proxy] = client

	return client, nil
}

// CloseIdleConnections releases the keep-alive connections held by every client
// in the pool. Without it, each discarded generation of the configuration would
// leave its idle connections open until the upstream or the OS closed them.
func (p *ClientPool) CloseIdleConnections() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, client := range p.clients {
		client.CloseIdleConnections()
	}
}

// Close implements io.Closer so the pool can take part in generation teardown.
func (p *ClientPool) Close() error {
	p.CloseIdleConnections()

	return nil
}

func newHTTPClient(proxy string) (*http.Client, error) {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errUnexpectedDefaultTransport
	}

	// Cloning the default transport keeps the standard library's timeouts,
	// connection limits and HTTP/2 support instead of silently opting out of all
	// of them by building a transport from its zero value.
	upstream := transport.Clone()
	upstream.MaxIdleConnsPerHost = maxIdleConnsPerHost
	upstream.ResponseHeaderTimeout = responseHeaderTimeout
	upstream.ForceAttemptHTTP2 = true

	if proxy != "" {
		parsedURL, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("failed to create http client: %w", err)
		}

		upstream.Proxy = http.ProxyURL(parsedURL)
	}

	return &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: upstream,
	}, nil
}
