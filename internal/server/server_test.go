package server_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/events"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/samber/lo"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	const porstCount = 5

	expectedContent := "Test"
	handler := infra.CastToHTTPHandler(infra.HandlerFunc(func(w contracts.ResponseWriter, _ *contracts.Request) error {
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, expectedContent)
		assert.NoError(t, err)

		return nil
	}))

	assertResponse := func(t *testing.T, url string, pool *x509.CertPool) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		require.NoError(t, err)

		var client *http.Client

		if pool != nil {
			client = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						MinVersion: tls.VersionTLS13,
						RootCAs:    pool,
						ServerName: "127.0.0.1",
					},
				},
			}
		} else {
			client = http.DefaultClient
		}

		response, err := client.Do(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode)

		bodyData, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		assert.Equal(t, expectedContent, string(bodyData))
	}

	t.Run("multiple http ports", func(t *testing.T) {
		freePorts := testutils.GetFreePorts(t, porstCount)

		targets := lo.Map(freePorts, func(port, _ int) server.Target {
			return server.Target{
				Address: hosts.Loopback.Port(port).String(),
				Handler: handler,
			}
		})

		manager := server.NewHostCertManager(afero.NewOsFs(), nil)
		instance := server.New(manager, events.NewBus())
		require.NoError(t, instance.Start(t.Context(), targets))

		defer func() {
			require.NoError(t, instance.Close())
		}()

		for _, port := range freePorts {
			t.Run(fmt.Sprintf("port %d", port), func(t *testing.T) {
				assertResponse(t, hosts.Loopback.HTTPPort(port).String(), nil)
			})
		}
	})

	t.Run("multiple https ports", func(t *testing.T) {
		freePorts := testutils.GetFreePorts(t, porstCount)

		fakeHome := t.TempDir()
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

		caDir := filepath.Join(fakeHome, ".config", "uncors")
		certPath, keyPath, err := server.GenerateCA(server.CAConfig{
			ValidityDays: 10,
			Fs:           fs,
			OutputDir:    caDir,
		})
		require.NoError(t, err)

		caCert, _, err := server.LoadCA(fs, certPath, keyPath)
		require.NoError(t, err)

		pool := x509.NewCertPool()
		pool.AddCert(caCert)

		targets := lo.Map(freePorts, func(port, _ int) server.Target {
			return server.Target{
				Address:   hosts.Loopback.Port(port).String(),
				Handler:   handler,
				EnableTLS: true,
			}
		})

		manager := server.NewHostCertManager(fs, nil)
		instance := server.New(manager, events.NewBus())
		require.NoError(t, instance.Start(t.Context(), targets))

		defer func() {
			require.NoError(t, instance.Close())
		}()

		for _, port := range freePorts {
			t.Run(fmt.Sprintf("port %d", port), func(t *testing.T) {
				assertResponse(t, hosts.Loopback.HTTPSPort(port).String(), pool)
			})
		}
	})

	t.Run("mix of http and https ports", func(t *testing.T) {
		freeHTTPPorts := testutils.GetFreePorts(t, porstCount)

		freeHTTPSPorts := testutils.GetFreePorts(t, porstCount)

		fakeHome := t.TempDir()
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

		caDir := filepath.Join(fakeHome, ".config", "uncors")
		certPath, keyPath, err := server.GenerateCA(server.CAConfig{
			ValidityDays: 10,
			Fs:           fs,
			OutputDir:    caDir,
		})
		require.NoError(t, err)

		caCert, _, err := server.LoadCA(fs, certPath, keyPath)
		require.NoError(t, err)

		pool := x509.NewCertPool()
		pool.AddCert(caCert)

		httpTargets := lo.Map(freeHTTPPorts, func(port, _ int) server.Target {
			return server.Target{
				Address: hosts.Loopback.Port(port).String(),
				Handler: handler,
			}
		})

		httpsTargets := lo.Map(freeHTTPSPorts, func(port, _ int) server.Target {
			return server.Target{
				Address:   hosts.Loopback.Port(port).String(),
				Handler:   handler,
				EnableTLS: true,
			}
		})

		manager := server.NewHostCertManager(fs, nil)
		instance := server.New(manager, events.NewBus())
		require.NoError(t, instance.Start(t.Context(), append(httpTargets, httpsTargets...)))

		defer func() {
			require.NoError(t, instance.Close())
		}()

		for _, port := range freeHTTPSPorts {
			t.Run(fmt.Sprintf("https port %d", port), func(t *testing.T) {
				assertResponse(t, hosts.Loopback.HTTPSPort(port).String(), pool)
			})
		}

		for _, port := range freeHTTPPorts {
			t.Run(fmt.Sprintf("http port %d", port), func(t *testing.T) {
				assertResponse(t, hosts.Loopback.HTTPPort(port).String(), nil)
			})
		}
	})

	t.Run("shutdown", func(t *testing.T) {
		port := testutils.GetFreePort(t)

		manager := server.NewHostCertManager(afero.NewOsFs(), nil)
		instance := server.New(manager, events.NewBus())
		require.NoError(t, instance.Start(t.Context(), []server.Target{
			{
				Address: hosts.Loopback.Port(port).String(),
				Handler: handler,
			},
		}))

		defer func() {
			require.NoError(t, instance.Close())
		}()

		time.Sleep(50 * time.Millisecond)

		assertResponse(t, hosts.Loopback.HTTPPort(port).String(), nil)

		err := instance.Shutdown(t.Context())
		require.NoError(t, err)

		assert.True(t, testutils.IsPortFree(port))
	})

	t.Run("close", func(t *testing.T) {
		port := testutils.GetFreePort(t)

		manager := server.NewHostCertManager(afero.NewOsFs(), nil)
		instance := server.New(manager, events.NewBus())
		require.NoError(t, instance.Start(t.Context(), []server.Target{
			{
				Address: hosts.Loopback.Port(port).String(),
				Handler: handler,
			},
		}))

		defer func() {
			require.NoError(t, instance.Close())
		}()

		assertResponse(t, hosts.Loopback.HTTPPort(port).String(), nil)

		err := instance.Close()
		require.NoError(t, err)

		assert.True(t, testutils.IsPortFree(port))
	})

	t.Run("Restart", func(t *testing.T) {
		initial := testutils.GetFreePort(t)
		restarted := testutils.GetFreePort(t)
		manager := server.NewHostCertManager(afero.NewOsFs(), nil)
		instance := server.New(manager, events.NewBus())

		require.NoError(t, instance.Start(t.Context(), []server.Target{
			{
				Address: hosts.Loopback.Port(initial).String(),
				Handler: handler,
			},
		}))

		assertResponse(t, hosts.Loopback.HTTPPort(initial).String(), nil)
		require.True(t, testutils.IsPortFree(restarted))

		err := instance.Restart(t.Context(), []server.Target{
			{
				Address: hosts.Loopback.Port(restarted).String(),
				Handler: handler,
			},
		})
		require.NoError(t, err)

		assert.True(t, testutils.IsPortFree(initial))
		assertResponse(t, hosts.Loopback.HTTPPort(restarted).String(), nil)
	})

	t.Run("Start binds nothing when one port is taken", func(t *testing.T) {
		ports := testutils.GetFreePorts(t, 2)
		occupy(t, ports[1])

		instance := server.New(server.NewHostCertManager(afero.NewOsFs(), nil), events.NewBus())

		err := instance.Start(t.Context(), []server.Target{
			{Address: hosts.Loopback.Port(ports[0]).String(), Handler: handler},
			{Address: hosts.Loopback.Port(ports[1]).String(), Handler: handler},
		})
		require.Error(t, err)

		assert.True(t, testutils.IsPortFree(ports[0]))
	})

	t.Run("Restart onto a taken port serves the previous targets again", func(t *testing.T) {
		ports := testutils.GetFreePorts(t, 2)
		instance := server.New(server.NewHostCertManager(afero.NewOsFs(), nil), events.NewBus())

		require.NoError(t, instance.Start(t.Context(), []server.Target{
			{Address: hosts.Loopback.Port(ports[0]).String(), Handler: handler},
		}))

		defer func() {
			require.NoError(t, instance.Close())
		}()

		occupy(t, ports[1])

		err := instance.Restart(t.Context(), []server.Target{
			{Address: hosts.Loopback.Port(ports[1]).String(), Handler: handler},
		})
		require.Error(t, err)

		assertResponse(t, hosts.Loopback.HTTPPort(ports[0]).String(), nil)
	})

	t.Run("Restart is not blocked by a request that outlives the shutdown timeout", func(t *testing.T) {
		initial := testutils.GetFreePort(t)
		restarted := testutils.GetFreePort(t)
		publisher := &testutils.EventRecorder{}
		manager := server.NewHostCertManager(afero.NewOsFs(), nil)
		instance := server.New(manager, publisher)
		server.SetShutdownTimeout(instance, 100*time.Millisecond)

		entered := make(chan struct{})
		release := make(chan struct{})

		defer close(release)

		// Shutdown cancels request contexts, so only a handler that ignores its
		// context (a busy Lua script, a write blocked on a slow client) can
		// outlive it.
		stuck := infra.CastToHTTPHandler(infra.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) error {
			close(entered)
			<-release

			return nil
		}))

		require.NoError(t, instance.Start(t.Context(), []server.Target{
			{
				Address: hosts.Loopback.Port(initial).String(),
				Handler: stuck,
			},
		}))

		defer func() {
			require.NoError(t, instance.Close())
		}()

		go func() {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
				hosts.Loopback.HTTPPort(initial).String(), nil)
			if !assert.NoError(t, err) {
				return
			}

			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				_ = resp.Body.Close()
			}
		}()

		<-entered

		err := instance.Restart(t.Context(), []server.Target{
			{
				Address: hosts.Loopback.Port(restarted).String(),
				Handler: handler,
			},
		})
		require.NoError(t, err)

		assertResponse(t, hosts.Loopback.HTTPPort(restarted).String(), nil)
		assert.Equal(t, []events.Notice{{
			Level: events.NoticeWarning,
			Text:  "In-flight requests did not finish before restart and were closed",
		}}, publisher.Notices())
	})

	t.Run("wait", func(t *testing.T) {
		queue := testutils.QueueEvents{}

		port := testutils.GetFreePort(t)

		manager := server.NewHostCertManager(afero.NewOsFs(), nil)
		instance := server.New(manager, events.NewBus())

		queue.Track("server started")

		require.NoError(t, instance.Start(t.Context(), []server.Target{
			{
				Address: hosts.Loopback.Port(port).String(),
				Handler: infra.CastToHTTPHandler(infra.HandlerFunc(func(w contracts.ResponseWriter, _ *contracts.Request) error {
					queue.Track("handler trigered")

					w.WriteHeader(http.StatusOK)
					_, err := fmt.Fprint(w, expectedContent)
					assert.NoError(t, err)

					return nil
				})),
			},
		}))

		defer func() {
			require.NoError(t, instance.Close())
		}()

		var waitGroup sync.WaitGroup

		waitGroup.Go(func() {
			queue.Track("waiting started")

			instance.Wait()

			queue.Track("waiting finished")
		})

		assertResponse(t, hosts.Loopback.HTTPPort(port).String(), nil)

		waitGroup.Go(func() {
			queue.Track("shutdown trigered")

			err := instance.Shutdown(t.Context())
			assert.NoError(t, err)
		})

		waitGroup.Wait()

		assert.Equal(t, []string{
			"server started",
			"waiting started",
			"handler trigered",
			"shutdown trigered",
			"waiting finished",
		}, queue.List())
	})

	t.Run("returns error when port is already in use", func(t *testing.T) {
		port := testutils.GetFreePort(t)

		ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", hosts.Loopback.Port(port).String())
		require.NoError(t, err)
		t.Cleanup(func() { ln.Close() })

		manager := server.NewHostCertManager(afero.NewOsFs(), nil)
		instance := server.New(manager, events.NewBus())
		err = instance.Start(t.Context(), []server.Target{
			{
				Address: hosts.Loopback.Port(port).String(),
				Handler: handler,
			},
		})

		require.Error(t, err)
	})
}

func occupy(t *testing.T, port int) {
	t.Helper()

	lc := &net.ListenConfig{}

	listener, err := lc.Listen(t.Context(), "tcp4", hosts.Loopback.Port(port).String())
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })
}
