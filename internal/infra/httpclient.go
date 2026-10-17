package infra

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/evg4b/uncors/internal/telemetry"
)

const (
	dialTimeout           = 30 * time.Second
	keepAlive             = 30 * time.Second
	tlsHandshakeTimeout   = 10 * time.Second
	responseHeaderTimeout = 5 * time.Minute
	idleConnTimeout       = 90 * time.Second
)

func MakeHTTPClient(proxy string) *http.Client {
	dialer := &net.Dialer{Timeout: dialTimeout, KeepAlive: keepAlive}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ResponseHeaderTimeout: responseHeaderTimeout,
		IdleConnTimeout:       idleConnTimeout,
	}

	if proxy != "" {
		parsedURL, err := url.Parse(proxy)
		if err != nil {
			panic(fmt.Errorf("failed to create http client: %w", err))
		}

		transport.Proxy = http.ProxyURL(parsedURL)
	}

	return &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: telemetry.WrapTransport(transport),
	}
}
