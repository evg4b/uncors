//go:build integration

package integration

import (
	"context"
	"crypto/x509"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// caValidityDays must stay clear of the proxy's expiration warning threshold
// (7 days); below it HostCertManager refuses to serve TLS.
const (
	caValidityDays   = 30
	configFilePerm   = 0o600
	proxyReadyWait   = 5 * time.Second
	proxyPollTick    = 25 * time.Millisecond
	proxyDialTimeout = 100 * time.Millisecond
)

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

	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)

	const configPath = "/uncors-config.yaml"

	err = afero.WriteFile(fs, configPath, data, configFilePerm)
	require.NoError(t, err)

	go func() {
		// --interactive=false overrides the default (true) so the proxy runs
		// in headless mode and actually starts its TCP listeners.
		container := di.NewContainer(di.WithFs(fs), di.WithArgs([]string{"-c", configPath, "--interactive=false"}))
		defer testutils.Close(t, container)

		err = cli.RunUncors(t.Context(), container)
		assert.NoError(t, err)
	}()

	waitForMappings(t, cfg)

	return caCert
}

// waitForMappings polls until every mapped port is accepting TCP connections.
func waitForMappings(t *testing.T, cfg *config.UncorsConfig) {
	t.Helper()

	for _, mapping := range cfg.Mappings {
		if mapping.From.Port == "" {
			continue
		}

		addr := net.JoinHostPort("127.0.0.1", mapping.From.Port)
		deadline := time.Now().Add(proxyReadyWait)

		for time.Now().Before(deadline) {
			dialer := &net.Dialer{Timeout: proxyDialTimeout}

			conn, dialErr := dialer.DialContext(context.Background(), "tcp", addr)
			if dialErr == nil {
				conn.Close()

				break
			}

			time.Sleep(proxyPollTick)
		}

		if time.Now().After(deadline) {
			port, _ := strconv.Atoi(mapping.From.Port)
			t.Fatalf("proxy port %d did not become ready within %s", port, proxyReadyWait)
		}
	}
}
