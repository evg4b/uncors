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

// ProxyHarness runs uncors in-process. Running in-process (rather than as a
// subprocess) keeps tests fast, deterministic and coverage-friendly. The struct
// exposes the loopback HTTP/HTTPS listener ports for tests that target the proxy
// directly by IP; domain-based routes are reached through the Hosts resolver.
type ProxyHarness struct {
	caCert    *x509.Certificate
	HTTPPort  int
	HTTPSPort int
}

// CACert returns the proxy's dev CA certificate, which the client must trust.
func (p *ProxyHarness) CACert() *x509.Certificate {
	return p.caCert
}

// HTTPSURL builds an absolute https URL on the loopback proxy for the given path.
func (p *ProxyHarness) HTTPSURL(path string) string {
	return testutils.JoinPath(hosts.Loopback.HTTPSPort(p.HTTPSPort), path)
}

// HTTPURL builds an absolute http URL on the loopback proxy for the given path.
func (p *ProxyHarness) HTTPURL(path string) string {
	return testutils.JoinPath(hosts.Loopback.HTTPPort(p.HTTPPort), path)
}

// bootProxy generates a fresh dev CA into an in-memory filesystem at the exact
// path HostCertManager reads from, starts uncors with the given mappings, and
// registers shutdown with t.Cleanup. It returns the CA the client must trust.
func bootProxy(t *testing.T, mappings config.Mappings) *x509.Certificate {
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

	app := uncors.CreateUncors(fs, server.NewRequestTracker(), mocks.NoopOutput(), "integration-test")

	err = app.Start(t.Context(), &config.UncorsConfig{Mappings: mappings})
	require.NoError(t, err)

	t.Cleanup(func() { _ = app.Close() })

	return caCert
}
