//go:build integration

package harness

import (
	"crypto/x509"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// caValidityDays must stay well clear of the proxy's expiration warning
// threshold (7 days); below it HostCertManager refuses to serve TLS.
const caValidityDays = 30

// ProxyHarness runs uncors in-process with one HTTP and one HTTPS mapping, both
// forwarding to a single backend. Running in-process (rather than as a
// subprocess) keeps tests fast, deterministic and coverage-friendly.
type ProxyHarness struct {
	caCert    *x509.Certificate
	HTTPPort  int
	HTTPSPort int
}

// NewProxyHarness boots uncors and registers shutdown with t.Cleanup. It mints a
// dev CA into an in-memory filesystem at the exact path HostCertManager reads
// from, so the proxy serves HTTPS with a CA the test client can trust. decorate
// may add mocks/rewrites/cache to the forwarding mappings; pass nil for plain
// forwarding.
func NewProxyHarness(t *testing.T, backendURL string, decorate func(*config.Mapping)) *ProxyHarness {
	t.Helper()

	fs := afero.NewMemMapFs()

	caDir, err := server.GetCAPath()
	require.NoError(t, err)

	certPath, keyPath, err := server.GenerateCA(server.CAConfig{
		Fs:           fs,
		ValidityDays: caValidityDays,
		OutputDir:    caDir,
	})
	require.NoError(t, err)

	caCert, _, err := server.LoadCA(fs, certPath, keyPath)
	require.NoError(t, err)

	httpPort := testutils.GetFreePort(t)
	httpsPort := testutils.GetFreePort(t)

	httpMapping := config.Mapping{From: hosts.Loopback.HTTPPort(httpPort), To: backendURL}
	httpsMapping := config.Mapping{From: hosts.Loopback.HTTPSPort(httpsPort), To: backendURL}

	if decorate != nil {
		decorate(&httpMapping)
		decorate(&httpsMapping)
	}

	app := uncors.CreateUncors(fs, server.NewRequestTracker(), mocks.NoopOutput(), "integration-test")

	err = app.Start(t.Context(), &config.UncorsConfig{
		Mappings: config.Mappings{httpMapping, httpsMapping},
	})
	require.NoError(t, err)

	t.Cleanup(func() { _ = app.Close() })

	return &ProxyHarness{caCert: caCert, HTTPPort: httpPort, HTTPSPort: httpsPort}
}

// CACert returns the proxy's dev CA certificate, which the client must trust.
func (p *ProxyHarness) CACert() *x509.Certificate {
	return p.caCert
}

// HTTPSURL builds an absolute https URL on the proxy for the given path.
func (p *ProxyHarness) HTTPSURL(path string) string {
	return testutils.JoinPath(hosts.Loopback.HTTPSPort(p.HTTPSPort), path)
}

// HTTPURL builds an absolute http URL on the proxy for the given path.
func (p *ProxyHarness) HTTPURL(path string) string {
	return testutils.JoinPath(hosts.Loopback.HTTPPort(p.HTTPPort), path)
}
