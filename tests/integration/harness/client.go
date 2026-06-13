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
func NewClient(caCert *x509.Certificate) *http.Client {
	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
				RootCAs:    pool,
			},
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
