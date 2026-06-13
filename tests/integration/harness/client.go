//go:build integration

package harness

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
)

// NewClient returns an HTTP client that trusts ONLY the proxy's dev CA, so a
// successful HTTPS call proves the proxy presented a valid CA-signed leaf rather
// than the test skipping verification. Redirects are not followed, so each call
// maps 1:1 to a backend hit, which keeps request-count assertions meaningful.
//
// When hosts is non-nil its in-memory resolution is applied, letting requests to
// real domains reach the loopback proxy with their Host header and SNI intact.
func NewClient(caCert *x509.Certificate, hosts *Hosts) *http.Client {
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
