package di_test

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func proxyConfiguration(port int) *config.UncorsConfig {
	return &config.UncorsConfig{
		CacheConfig: config.CacheConfig{MaxSize: 100, ExpirationTime: time.Minute},
		Mappings: config.Mappings{
			{
				From: hosts.Localhost.HTTPPort(port),
				To:   hosts.Localhost.HTTP(),
			},
		},
	}
}

const dialTimeout = time.Second

func dial(t *testing.T, port int) (net.Conn, error) {
	t.Helper()

	dialer := &net.Dialer{Timeout: dialTimeout}

	return dialer.DialContext(t.Context(), "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
}

func assertListening(t *testing.T, port int) {
	t.Helper()

	conn, err := dial(t, port)
	require.NoError(t, err)
	require.NoError(t, conn.Close())
}

// occupy binds port for the duration of the test so the proxy cannot.
func occupy(t *testing.T, port int) {
	t.Helper()

	lc := &net.ListenConfig{}

	listener, err := lc.Listen(t.Context(), "tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })
}

func TestProxy(t *testing.T) {
	t.Run("is a container singleton", func(t *testing.T) {
		container := di.NewContainer(di.WithFs(afero.NewMemMapFs()))
		defer testutils.Close(t, container)

		first := container.Proxy()

		assert.Same(t, first, container.Proxy())
	})

	t.Run("serves the configured mapping", func(t *testing.T) {
		port := testutils.GetFreePort(t)

		container := di.NewContainer(di.WithFs(afero.NewMemMapFs()))
		defer testutils.Close(t, container)

		proxy := container.Proxy()

		require.NoError(t, proxy.Start(t.Context(), proxyConfiguration(port)))

		defer testutils.Close(t, proxy)

		assertListening(t, port)
	})

	t.Run("moves to the port of the reloaded config", func(t *testing.T) {
		ports := testutils.GetFreePorts(t, 2)

		container := di.NewContainer(di.WithFs(afero.NewMemMapFs()))
		defer testutils.Close(t, container)

		proxy := container.Proxy()

		require.NoError(t, proxy.Start(t.Context(), proxyConfiguration(ports[0])))

		defer testutils.Close(t, proxy)

		require.NoError(t, proxy.Restart(t.Context(), proxyConfiguration(ports[1])))

		assertListening(t, ports[1])
	})

	t.Run("stops serving the port the reloaded config dropped", func(t *testing.T) {
		ports := testutils.GetFreePorts(t, 2)

		container := di.NewContainer(di.WithFs(afero.NewMemMapFs()))
		defer testutils.Close(t, container)

		proxy := container.Proxy()

		require.NoError(t, proxy.Start(t.Context(), proxyConfiguration(ports[0])))

		defer testutils.Close(t, proxy)

		require.NoError(t, proxy.Restart(t.Context(), proxyConfiguration(ports[1])))

		assert.Eventually(t, func() bool {
			conn, err := dial(t, ports[0])
			if err != nil {
				return true
			}

			_ = conn.Close()

			return false
		}, time.Second, 20*time.Millisecond, "old port is still accepting connections")
	})

	t.Run("restart returns an error when the new port is taken", func(t *testing.T) {
		ports := testutils.GetFreePorts(t, 2)

		container := di.NewContainer(di.WithFs(afero.NewMemMapFs()))
		defer testutils.Close(t, container)

		proxy := container.Proxy()

		require.NoError(t, proxy.Start(t.Context(), proxyConfiguration(ports[0])))

		defer testutils.Close(t, proxy)

		occupy(t, ports[1])

		require.Error(t, proxy.Restart(t.Context(), proxyConfiguration(ports[1])))
	})

	t.Run("returns an error when the port is already taken", func(t *testing.T) {
		port := testutils.GetFreePort(t)

		occupy(t, port)

		container := di.NewContainer(di.WithFs(afero.NewMemMapFs()))
		defer testutils.Close(t, container)

		require.Error(t, container.Proxy().Start(t.Context(), proxyConfiguration(port)))
	})

	t.Run("shutdown releases the active generation and is idempotent", func(t *testing.T) {
		port := testutils.GetFreePort(t)

		container := di.NewContainer(di.WithFs(afero.NewMemMapFs()))
		defer testutils.Close(t, container)

		proxy := container.Proxy()

		require.NoError(t, proxy.Start(t.Context(), proxyConfiguration(port)))

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		require.NoError(t, proxy.Shutdown(ctx))
		require.NoError(t, proxy.Shutdown(ctx))

		proxy.Wait()
	})
}
