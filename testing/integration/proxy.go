//go:build integration

package integration

import (
	"context"
	"crypto/x509"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// caValidityDays must stay clear of the proxy's expiration warning threshold
// (7 days); below it HostCertManager refuses to serve TLS.
const (
	caValidityDays   = 30
	configFilePerm   = 0o600
	configPath       = "/uncors-config.yaml"
	proxyReadyWait   = 5 * time.Second
	proxyPollTick    = 25 * time.Millisecond
	proxyDialTimeout = 100 * time.Millisecond
)

// bootProxy generates a fresh dev CA, writes cfg to the test filesystem and
// starts uncors through the same cli.RunUncors entry point production uses.
// Returns the CA that the client must trust to complete TLS handshakes.
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

	require.NoError(t, afero.WriteFile(fs, configPath, marshalConfig(t, cfg), configFilePerm))

	go func() {
		// --interactive=false overrides the default (true) so the proxy runs
		// in headless mode and actually starts its TCP listeners.
		container := di.NewContainer(di.WithFs(fs), di.WithArgs([]string{"-c", configPath, "--interactive=false"}))
		defer testutils.Close(t, container)

		// The test goroutine may already have finished by the time RunUncors
		// returns, so failures are logged rather than asserted.
		runErr := cli.RunUncors(t.Context(), container)
		if runErr != nil {
			t.Logf("RunUncors returned: %v", runErr)
		}
	}()

	waitForMappings(t, cfg)

	return caCert
}

// marshalConfig serialises cfg to YAML. Inline scripts are trimmed first:
// gopkg.in/yaml.v3 emits a string starting with a newline as a "|4" block
// scalar whose content indentation does not parse back.
func marshalConfig(t *testing.T, cfg *config.UncorsConfig) []byte {
	t.Helper()

	normalised := *cfg
	normalised.Mappings = make(config.Mappings, 0, len(cfg.Mappings))

	for _, mapping := range cfg.Mappings {
		mapping = mapping.Clone()
		for i := range mapping.Scripts {
			mapping.Scripts[i].Script = strings.TrimSpace(mapping.Scripts[i].Script)
		}

		normalised.Mappings = append(normalised.Mappings, mapping)
	}

	data, err := yaml.Marshal(&normalised)
	require.NoError(t, err)

	return data
}

// waitForMappings blocks until every mapped port is accepting TCP connections.
func waitForMappings(t *testing.T, cfg *config.UncorsConfig) {
	t.Helper()

	for _, mapping := range cfg.Mappings {
		if mapping.From.Port == "" {
			continue
		}

		waitForPort(t, net.JoinHostPort("127.0.0.1", mapping.From.Port))
	}
}

func waitForPort(t *testing.T, addr string) {
	t.Helper()

	dialer := &net.Dialer{Timeout: proxyDialTimeout}
	deadline := time.Now().Add(proxyReadyWait)

	for time.Now().Before(deadline) {
		conn, err := dialer.DialContext(context.Background(), "tcp", addr)
		if err == nil {
			conn.Close()

			return
		}

		time.Sleep(proxyPollTick)
	}

	t.Fatalf("proxy address %s did not become ready within %s", addr, proxyReadyWait)
}
