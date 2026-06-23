//go:build integration

package integration

import (
	"crypto/x509"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/server"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// caValidityDays must stay clear of the proxy's expiration warning threshold
// (7 days); below it HostCertManager refuses to serve TLS.
const caValidityDays = 30

// bootProxy generates a fresh dev CA, starts uncors in-process with the given
// config, and registers shutdown with t.Cleanup. Returns the CA that the client
// must trust to complete TLS handshakes with the proxy.
func bootProxy(t *testing.T, fs afero.Fs, cfg *config.UncorsConfig) *x509.Certificate {
	t.Helper()

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

	container := di.NewContainer(di.WithFs(fs))

	targets, err := container.Targets(cfg)
	require.NoError(t, err)

	srv := container.Server()

	err = srv.Start(t.Context(), targets)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = srv.Close()
		_ = container.Close()
	})

	return caCert
}
