//go:build integration

package integration

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
)

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
