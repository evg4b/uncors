//go:build integration

package integration

import (
	"crypto/x509"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/server"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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

	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)

	const configPath = "/uncors-config.yaml"

	err = afero.WriteFile(fs, configPath, data, 0o644)
	require.NoError(t, err)

	go func() {
		_ = cli.RunUncors(t.Context(), fs, []string{"-c", configPath})
	}()

	waitForMappings(t, cfg)

	return caCert
}

// waitForMappings polls until every mapped port is accepting TCP connections.
func waitForMappings(t *testing.T, cfg *config.UncorsConfig) {
	t.Helper()

	const (
		readyTimeout  = 5 * time.Second
		pollInterval  = 25 * time.Millisecond
		dialTimeout   = 100 * time.Millisecond
	)

	for _, m := range cfg.Mappings {
		if m.From.Port == "" {
			continue
		}

		addr := net.JoinHostPort("127.0.0.1", m.From.Port)
		deadline := time.Now().Add(readyTimeout)

		for time.Now().Before(deadline) {
			conn, dialErr := net.DialTimeout("tcp", addr, dialTimeout)
			if dialErr == nil {
				conn.Close()

				break
			}

			time.Sleep(pollInterval)
		}

		if time.Now().After(deadline) {
			port, _ := strconv.Atoi(m.From.Port)
			t.Fatalf("proxy port %d did not become ready within %s", port, readyTimeout)
		}
	}
}
