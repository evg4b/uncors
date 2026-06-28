package cli_test

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// httpMapping builds a minimal valid UncorsConfig for an HTTP proxy on a free port.
func httpMapping(t *testing.T) (*config.UncorsConfig, int) {
	t.Helper()

	port := testutils.GetFreePort(t)

	cfg := &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Localhost.HTTPPort(port),
			To:   hosts.Localhost.HTTP(),
		}},
		CacheConfig: config.CacheConfig{
			ExpirationTime: config.DefaultExpirationTime,
			MaxSize:        config.DefaultMaxSize,
			Methods:        []string{http.MethodGet},
		},
	}

	return cfg, port
}

// waitForPort blocks until the TCP address accepts connections or times out.
func waitForPort(t *testing.T, addr string) {
	t.Helper()

	const (
		dialTimeout = 100 * time.Millisecond
		pollTick    = 25 * time.Millisecond
		readyWait   = 5 * time.Second
	)

	deadline := time.Now().Add(readyWait)

	for time.Now().Before(deadline) {
		dialer := &net.Dialer{Timeout: dialTimeout}

		conn, err := dialer.DialContext(context.Background(), "tcp", addr)
		if err == nil {
			conn.Close()

			return
		}

		time.Sleep(pollTick)
	}

	t.Fatal("port did not become ready within 5s: " + addr)
}

// writeConfig marshals cfg to YAML and writes it to path on the real OS filesystem.
func writeConfig(t *testing.T, path string, cfg *config.UncorsConfig) {
	t.Helper()

	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))
}

// startProxy starts RunUncors in a goroutine and returns a channel that
// receives the error when it exits. The caller must cancel the context and
// drain the channel to ensure the goroutine has fully stopped.
func startProxy(ctx context.Context, fs afero.Fs, args []string) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		container := di.NewContainer(di.WithFs(fs), di.WithArgs(args))
		defer func() {
			errCh <- container.Close()
		}()

		errCh <- cli.RunUncors(ctx, container)
	}()

	return errCh
}

func TestRunUncors(t *testing.T) {
	t.Run("returns error when LoadConfiguration fails", func(t *testing.T) {
		// No --from/--to flags and no config file → "mappings must not be empty"
		container := di.NewContainer(di.WithArgs([]string{}))
		defer testutils.Close(t, container)

		err := cli.RunUncors(context.Background(), container)
		require.Error(t, err)
	})

	t.Run("returns nil for --version flag", func(t *testing.T) {
		container := di.NewContainer(di.WithArgs([]string{"--version"}))
		defer testutils.Close(t, container)

		err := cli.RunUncors(context.Background(), container)
		require.NoError(t, err)
	})

	t.Run("returns nil for --help flag", func(t *testing.T) {
		container := di.NewContainer(di.WithArgs([]string{"--help"}))
		defer testutils.Close(t, container)

		err := cli.RunUncors(context.Background(), container)
		require.NoError(t, err)
	})

	t.Run("non-interactive: starts server and shuts down on context cancellation", func(t *testing.T) {
		cfg, port := httpMapping(t)
		fs := afero.NewMemMapFs()

		data, err := yaml.Marshal(cfg)
		require.NoError(t, err)
		require.NoError(t, afero.WriteFile(fs, "/config.yaml", data, 0o600))

		ctx, cancel := context.WithCancel(context.Background())

		errCh := startProxy(ctx, fs, []string{"-c", "/config.yaml", "--interactive=false"})

		waitForPort(t, net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		cancel()

		select {
		case err := <-errCh:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Fatal("RunUncors did not exit after context cancellation")
		}
	})

	t.Run("non-interactive: returns error when port is already in use", func(t *testing.T) {
		cfg, port := httpMapping(t)

		// Occupy the port so srv.Start fails.
		lc := &net.ListenConfig{}

		listener, err := lc.Listen(context.Background(), "tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		require.NoError(t, err)

		defer listener.Close()

		fs := afero.NewMemMapFs()

		data, err := yaml.Marshal(cfg)
		require.NoError(t, err)
		require.NoError(t, afero.WriteFile(fs, "/config.yaml", data, 0o600))

		container := di.NewContainer(di.WithFs(fs), di.WithArgs([]string{"-c", "/config.yaml", "--interactive=false"}))
		defer testutils.Close(t, container)

		err = cli.RunUncors(context.Background(), container)
		require.Error(t, err)
	})

	t.Run("non-interactive: reloads valid config on file change", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.yaml")

		cfg, port := httpMapping(t)
		writeConfig(t, configPath, cfg)

		ctx, cancel := context.WithCancel(context.Background())

		errCh := startProxy(ctx, afero.NewOsFs(), []string{"-c", configPath, "--interactive=false"})

		waitForPort(t, net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))

		// Overwrite with same valid config — watcher fires, reloadServer runs.
		writeConfig(t, configPath, cfg)

		// Let the debounce + reload settle before stopping.
		time.Sleep(200 * time.Millisecond)

		cancel()

		select {
		case err := <-errCh:
			if err != nil {
				t.Logf("RunUncors returned error: %v", err)
			}

			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Fatal("RunUncors did not exit after context cancellation")
		}
	})

	t.Run("non-interactive: logs error when config reload produces invalid config", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.yaml")

		cfg, port := httpMapping(t)
		writeConfig(t, configPath, cfg)

		ctx, cancel := context.WithCancel(context.Background())

		errCh := startProxy(ctx, afero.NewOsFs(), []string{"-c", configPath, "--interactive=false"})

		waitForPort(t, net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))

		// Write an invalid config (empty mappings) — reloadServer returns an error.
		require.NoError(t, os.WriteFile(configPath, []byte("mappings: []\n"), 0o600))

		// Let the debounce + reload settle before stopping.
		time.Sleep(200 * time.Millisecond)

		cancel()

		select {
		case err := <-errCh:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Fatal("RunUncors did not exit after context cancellation")
		}
	})
}
