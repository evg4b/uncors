//go:build integration

package integration

import (
	"context"
	"crypto/x509"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/engine"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/server"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// caValidityDays must stay clear of the proxy's expiration warning threshold
// (7 days); below it HostCertManager refuses to serve TLS.
const (
	caValidityDays    = 30
	configFilePerm    = 0o600
	configPath        = "/uncors-config.yaml"
	proxyReadyWait    = 5 * time.Second
	proxyPollTick     = 25 * time.Millisecond
	proxyDialTimeout  = 100 * time.Millisecond
	proxyShutdownWait = 5 * time.Second
)

func bootProxy(t *testing.T, fs afero.Fs, cfg *config.UncorsConfig) (*x509.Certificate, *eventRecorder) {
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

	loaded, _, err := config.LoadConfiguration(t.Context(), fs, nil, []string{"-c", configPath})
	require.NoError(t, err)

	bus := events.NewBus()
	recorder := recordEvents(bus.Subscribe(eventsBuffer))
	proxy := engine.New(engine.WithFs(fs), engine.WithPublisher(bus))

	require.NoError(t, proxy.Start(t.Context(), loaded))

	// Shutting down closes the generation, which flushes HAR files; cleanups
	// run in reverse order, so this happens before temp dirs are removed.
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), proxyShutdownWait)
		defer cancel()

		assert.NoError(t, proxy.Shutdown(ctx))
		proxy.Wait()
		bus.Close()
		<-recorder.done
	})

	waitForMappings(t, cfg)

	return caCert, recorder
}

// Scripts are trimmed: yaml.v3 emits a leading newline as a "|4" block scalar
// that does not parse back.
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
