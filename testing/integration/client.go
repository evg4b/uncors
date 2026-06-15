//go:build integration

package integration

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
)

// newClient returns an HTTP client that trusts only the proxy's dev CA.
// TLS verification is strict (not skipped), so a successful HTTPS request
// proves the proxy presented a CA-signed leaf. Redirects are not followed so
// each call maps 1:1 to a backend hit.
func newClient(caCert *x509.Certificate, hosts *Hosts) *http.Client {
	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
			RootCAs:    pool,
		},
	}

	if hosts != nil {
		transport.DialContext = hosts.DialContext
	}

	return &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
