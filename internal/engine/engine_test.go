package engine_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/certs"
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

		proxy := engine.New(afero.NewMemMapFs())

		require.NoError(t, proxy.Start(t.Context(), proxyConfiguration(port)))

		defer testutils.Close(t, proxy)

		assertListening(t, port)
	})

	t.Run("moves to the port of the reloaded config", func(t *testing.T) {
		ports := testutils.GetFreePorts(t, 2)

		proxy := engine.New(afero.NewMemMapFs())

		require.NoError(t, proxy.Start(t.Context(), proxyConfiguration(ports[0])))

		defer testutils.Close(t, proxy)

		require.NoError(t, proxy.Restart(t.Context(), proxyConfiguration(ports[1])))

		assertListening(t, ports[1])
	})

	t.Run("stops serving the port the reloaded config dropped", func(t *testing.T) {
		ports := testutils.GetFreePorts(t, 2)

		proxy := engine.New(afero.NewMemMapFs())

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

	t.Run("concurrent restarts leave exactly one generation serving", func(t *testing.T) {
		const restarts = 8

		ports := testutils.GetFreePorts(t, restarts+1)

		proxy := engine.New(afero.NewMemMapFs())

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

		proxy := engine.New(afero.NewMemMapFs())

		require.NoError(t, proxy.Start(t.Context(), proxyConfiguration(ports[0])))

		defer testutils.Close(t, proxy)

		occupy(t, ports[1])

		require.Error(t, proxy.Restart(t.Context(), proxyConfiguration(ports[1])))

		assertListening(t, ports[0])
	})

	t.Run("a failed restart binds none of the new ports", func(t *testing.T) {
		ports := testutils.GetFreePorts(t, 3)

		proxy := engine.New(afero.NewMemMapFs())

		require.NoError(t, proxy.Start(t.Context(), proxyConfiguration(ports[0])))

		defer testutils.Close(t, proxy)

		occupy(t, ports[2])

		cfg := proxyConfiguration(ports[1])
		cfg.Mappings = append(cfg.Mappings, proxyConfiguration(ports[2]).Mappings...)

		require.Error(t, proxy.Restart(t.Context(), cfg))

		assertListening(t, ports[0])
		assert.True(t, testutils.IsPortFree(ports[1]))
	})

	t.Run("returns an error when the port is already taken", func(t *testing.T) {
		port := testutils.GetFreePort(t)

		occupy(t, port)

		proxy := engine.New(afero.NewMemMapFs())

		require.Error(t, proxy.Start(t.Context(), proxyConfiguration(port)))
	})

	t.Run("shutdown releases the active generation and is idempotent", func(t *testing.T) {
		port := testutils.GetFreePort(t)

		proxy := engine.New(afero.NewMemMapFs())

		require.NoError(t, proxy.Start(t.Context(), proxyConfiguration(port)))

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		require.NoError(t, proxy.Shutdown(ctx))
		require.NoError(t, proxy.Shutdown(ctx))

		proxy.Wait()
	})
}

func TestEngineClosesIdleUpstreamConnectionsOnRestart(t *testing.T) {
	closed := make(chan struct{})

	var once sync.Once

	upstream := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	upstream.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateClosed {
			once.Do(func() { close(closed) })
		}
	}

	upstream.Start()
	defer upstream.Close()

	port := testutils.GetFreePort(t)
	cfg := proxyConfiguration(port)
	cfg.Mappings[0].To = hosts.Parse(upstream.URL)

	proxy := engine.New(afero.NewMemMapFs())
	require.NoError(t, proxy.Start(t.Context(), cfg))

	defer testutils.Close(t, proxy)

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
		hosts.Localhost.HTTPPort(port).String()+"/", nil)
	require.NoError(t, err)

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusOK, response.StatusCode)

	require.NoError(t, proxy.Restart(t.Context(), cfg))

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("the idle upstream connection of the previous generation was not closed")
	}
}

func TestEnginePublishesLifecycleEvents(t *testing.T) {
	ports := testutils.GetFreePorts(t, 3)

	bus := events.NewBus()
	subscription := bus.Subscribe(10)

	proxy := engine.New(afero.NewMemMapFs(), engine.WithPublisher(bus))
	defer testutils.Close(t, proxy)

	first := proxyConfiguration(ports[0])
	require.NoError(t, proxy.Start(t.Context(), first))
	assert.Equal(t, events.ServerStarted{Mappings: first.Mappings}, <-subscription.Events())

	second := proxyConfiguration(ports[1])
	require.NoError(t, proxy.Restart(t.Context(), second))
	assert.Equal(t, events.ServerRestarted{Mappings: second.Mappings}, <-subscription.Events())

	// A port the engine never used: the one it just left may not be free yet.
	occupy(t, ports[2])
	require.Error(t, proxy.Restart(t.Context(), proxyConfiguration(ports[2])))

	select {
	case event := <-subscription.Events():
		assert.Failf(t, "unexpected event", "a failed restart published %#v", event)
	default:
	}
}

func httpsConfiguration(host string, port int) *config.UncorsConfig {
	return &config.UncorsConfig{
		CacheConfig: config.CacheConfig{MaxSize: 100, ExpirationTime: time.Minute},
		Mappings: config.Mappings{
			{
				From: hosts.Parse("https://" + net.JoinHostPort(host, strconv.Itoa(port))),
				To:   hosts.Localhost.HTTP(),
			},
		},
	}
}

func handshake(t *testing.T, pool *x509.CertPool, port int, serverName string) error {
	t.Helper()

	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: dialTimeout},
		Config: &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    pool,
			ServerName: serverName,
		},
	}

	conn, err := dialer.DialContext(t.Context(), "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return err
	}

	return conn.Close()
}

func TestEngineIssuesCertificatesOnlyForMappedHosts(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	fs := afero.NewMemMapFs()

	caDir, err := certs.Dir()
	require.NoError(t, err)

	certPath, keyPath, err := certs.Generate(fs, caDir, 30)
	require.NoError(t, err)

	caCert, _, err := certs.Load(fs, certPath, keyPath)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	ports := testutils.GetFreePorts(t, 3)

	proxy := engine.New(fs)
	require.NoError(t, proxy.Start(t.Context(), httpsConfiguration("one.local", ports[0])))

	defer testutils.Close(t, proxy)

	require.NoError(t, handshake(t, pool, ports[0], "one.local"))
	require.Error(t, handshake(t, pool, ports[0], "two.local"))

	require.NoError(t, proxy.Restart(t.Context(), httpsConfiguration("two.local", ports[1])))

	require.NoError(t, handshake(t, pool, ports[1], "two.local"))
	require.Error(t, handshake(t, pool, ports[1], "one.local"))

	occupy(t, ports[2])
	require.Error(t, proxy.Restart(t.Context(), httpsConfiguration("three.local", ports[2])))

	require.NoError(t, handshake(t, pool, ports[1], "two.local"))
	require.Error(t, handshake(t, pool, ports[1], "three.local"))
}
