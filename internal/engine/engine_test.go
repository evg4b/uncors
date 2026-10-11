package engine_test

import (
	"context"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/engine"
	"github.com/evg4b/uncors/internal/events"
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

// occupy binds port for the duration of the test so the engine cannot.
func occupy(t *testing.T, port int) {
	t.Helper()

	lc := &net.ListenConfig{}

	listener, err := lc.Listen(t.Context(), "tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })
}

func TestEngine(t *testing.T) {
	t.Run("serves the configured mapping", func(t *testing.T) {
		port := testutils.GetFreePort(t)

		proxy := engine.New(engine.WithFs(afero.NewMemMapFs()))

		require.NoError(t, proxy.Start(t.Context(), proxyConfiguration(port)))

		defer testutils.Close(t, proxy)

		assertListening(t, port)
	})

	t.Run("moves to the port of the reloaded config", func(t *testing.T) {
		ports := testutils.GetFreePorts(t, 2)

		proxy := engine.New(engine.WithFs(afero.NewMemMapFs()))

		require.NoError(t, proxy.Start(t.Context(), proxyConfiguration(ports[0])))

		defer testutils.Close(t, proxy)

		require.NoError(t, proxy.Restart(t.Context(), proxyConfiguration(ports[1])))

		assertListening(t, ports[1])
	})

	t.Run("stops serving the port the reloaded config dropped", func(t *testing.T) {
		ports := testutils.GetFreePorts(t, 2)

		proxy := engine.New(engine.WithFs(afero.NewMemMapFs()))

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

	// Restarts may be requested from several goroutines at the same time. Each
	// must replace the previous generation completely, never leave an orphaned
	// listener behind.
	t.Run("concurrent restarts leave exactly one generation serving", func(t *testing.T) {
		const restarts = 8

		ports := testutils.GetFreePorts(t, restarts+1)

		proxy := engine.New(engine.WithFs(afero.NewMemMapFs()))

		require.NoError(t, proxy.Start(t.Context(), proxyConfiguration(ports[0])))
		defer testutils.Close(t, proxy)

		var waitGroup sync.WaitGroup
		for _, port := range ports[1:] {
			waitGroup.Go(func() {
				assert.NoError(t, proxy.Restart(t.Context(), proxyConfiguration(port)))
			})
		}

		waitGroup.Wait()

		listening := 0

		for _, port := range ports {
			conn, err := dial(t, port)
			if err == nil {
				listening++

				_ = conn.Close()
			}
		}

		assert.Equal(t, 1, listening, "exactly one generation must be listening")
	})

	t.Run("restart returns an error when the new port is taken", func(t *testing.T) {
		ports := testutils.GetFreePorts(t, 2)

		proxy := engine.New(engine.WithFs(afero.NewMemMapFs()))

		require.NoError(t, proxy.Start(t.Context(), proxyConfiguration(ports[0])))

		defer testutils.Close(t, proxy)

		occupy(t, ports[1])

		require.Error(t, proxy.Restart(t.Context(), proxyConfiguration(ports[1])))
	})

	t.Run("returns an error when the port is already taken", func(t *testing.T) {
		port := testutils.GetFreePort(t)

		occupy(t, port)

		proxy := engine.New(engine.WithFs(afero.NewMemMapFs()))

		require.Error(t, proxy.Start(t.Context(), proxyConfiguration(port)))
	})

	t.Run("shutdown releases the active generation and is idempotent", func(t *testing.T) {
		port := testutils.GetFreePort(t)

		proxy := engine.New(engine.WithFs(afero.NewMemMapFs()))

		require.NoError(t, proxy.Start(t.Context(), proxyConfiguration(port)))

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		require.NoError(t, proxy.Shutdown(ctx))
		require.NoError(t, proxy.Shutdown(ctx))

		proxy.Wait()
	})
}

func TestEnginePublishesLifecycleEvents(t *testing.T) {
	ports := testutils.GetFreePorts(t, 2)

	bus := events.NewBus()
	subscription := bus.Subscribe(10)

	proxy := engine.New(engine.WithFs(afero.NewMemMapFs()), engine.WithPublisher(bus))
	defer testutils.Close(t, proxy)

	first := proxyConfiguration(ports[0])
	require.NoError(t, proxy.Start(t.Context(), first))
	assert.Equal(t, events.ServerStarted{Mappings: first.Mappings}, <-subscription.Events())

	second := proxyConfiguration(ports[1])
	require.NoError(t, proxy.Restart(t.Context(), second))
	assert.Equal(t, events.ServerRestarted{Mappings: second.Mappings}, <-subscription.Events())

	occupy(t, ports[0])
	require.Error(t, proxy.Restart(t.Context(), proxyConfiguration(ports[0])))

	select {
	case event := <-subscription.Events():
		assert.Failf(t, "unexpected event", "a failed restart published %#v", event)
	default:
	}
}
