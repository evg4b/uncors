package cli_test

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

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

func writeConfig(t *testing.T, path string, cfg *config.UncorsConfig) {
	t.Helper()

	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))
}

type syncBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buffer.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return ansi.Strip(b.buffer.String())
}

func env(fs afero.Fs, args ...string) cli.Env {
	return cli.Env{Fs: fs, Stdout: io.Discard, Args: args, Version: "x.x.x"}
}

func startProxy(ctx context.Context, env cli.Env) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- cli.RunUncors(ctx, env)
	}()

	return errCh
}

func TestRunUncors(t *testing.T) {
	t.Run("returns error when LoadConfiguration fails", func(t *testing.T) {
		err := cli.RunUncors(context.Background(), env(afero.NewMemMapFs()))
		require.Error(t, err)
	})

	t.Run("prints the version for --version flag", func(t *testing.T) {
		output := &syncBuffer{}
		versionEnv := env(afero.NewMemMapFs(), "--version")
		versionEnv.Stdout = output
		versionEnv.Version = "1.2.3"

		err := cli.RunUncors(context.Background(), versionEnv)
		require.NoError(t, err)
		assert.Equal(t, "1.2.3\n", output.String())
	})

	t.Run("returns nil for --help flag", func(t *testing.T) {
		err := cli.RunUncors(context.Background(), env(afero.NewMemMapFs(), "--help"))
		require.NoError(t, err)
	})

	t.Run("non-interactive: starts server and shuts down on context cancellation", func(t *testing.T) {
		cfg, port := httpMapping(t)
		fs := afero.NewMemMapFs()

		data, err := yaml.Marshal(cfg)
		require.NoError(t, err)
		require.NoError(t, afero.WriteFile(fs, "/config.yaml", data, 0o600))

		ctx, cancel := context.WithCancel(context.Background())

		output := &syncBuffer{}
		runEnv := env(fs, "-c", "/config.yaml", "--interactive=false")
		runEnv.Stdout = output

		errCh := startProxy(ctx, runEnv)

		waitForPort(t, net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		cancel()

		select {
		case err := <-errCh:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Fatal("RunUncors did not exit after context cancellation")
		}

		assert.Contains(t, output.String(), "DON'T USE IT FOR PRODUCTION!")
		assert.Contains(t, output.String(), "Server was stopped")
	})

	t.Run("non-interactive: returns error when port is already in use", func(t *testing.T) {
		cfg, port := httpMapping(t)

		lc := &net.ListenConfig{}

		listener, err := lc.Listen(context.Background(), "tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		require.NoError(t, err)

		defer listener.Close()

		fs := afero.NewMemMapFs()

		data, err := yaml.Marshal(cfg)
		require.NoError(t, err)
		require.NoError(t, afero.WriteFile(fs, "/config.yaml", data, 0o600))

		err = cli.RunUncors(context.Background(), env(fs, "-c", "/config.yaml", "--interactive=false"))
		require.Error(t, err)
	})

	t.Run("non-interactive: reloads valid config on file change", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.yaml")

		cfg, port := httpMapping(t)
		writeConfig(t, configPath, cfg)

		ctx, cancel := context.WithCancel(context.Background())

		output := &syncBuffer{}
		runEnv := env(afero.NewOsFs(), "-c", configPath, "--interactive=false")
		runEnv.Stdout = output

		errCh := startProxy(ctx, runEnv)

		waitForPort(t, net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))

		writeConfig(t, configPath, cfg)

		assert.Eventually(t, func() bool {
			return strings.Contains(output.String(), "Server restarted")
		}, 5*time.Second, 20*time.Millisecond)

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

	t.Run("non-interactive: keeps the previous config when the reload is invalid", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.yaml")

		cfg, port := httpMapping(t)
		writeConfig(t, configPath, cfg)

		ctx, cancel := context.WithCancel(context.Background())

		output := &syncBuffer{}
		runEnv := env(afero.NewOsFs(), "-c", configPath, "--interactive=false")
		runEnv.Stdout = output

		errCh := startProxy(ctx, runEnv)

		waitForPort(t, net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))

		require.NoError(t, os.WriteFile(configPath, []byte("mappings: []\n"), 0o600))

		assert.Eventually(t, func() bool {
			return strings.Contains(output.String(), "Failed to reload configuration, keeping the previous one")
		}, 5*time.Second, 20*time.Millisecond)

		waitForPort(t, net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))

		cancel()

		select {
		case err := <-errCh:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Fatal("RunUncors did not exit after context cancellation")
		}
	})
}
