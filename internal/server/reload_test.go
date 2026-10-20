package server_test

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/certs"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func respond(body string) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, body)
	})
}

type blocking struct {
	entered  chan struct{}
	release  chan struct{}
	stubborn bool
}

func newBlocking() *blocking {
	return &blocking{entered: make(chan struct{}), release: make(chan struct{})}
}

func (b *blocking) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	close(b.entered)

	if b.stubborn {
		<-b.release
	} else {
		select {
		case <-b.release:
		case <-request.Context().Done():
			_, _ = io.WriteString(writer, "cancelled")

			return
		}
	}

	_, _ = io.WriteString(writer, "slow")
}

func get(t *testing.T, client *http.Client, url string) (string, error) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", err
	}

	response, err := client.Do(request)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	return string(body), err
}

func newClient() *http.Client {
	return &http.Client{Transport: &http.Transport{}}
}

func newInstance(t *testing.T) *server.Server {
	t.Helper()

	instance := server.New(certs.NewManager(afero.NewOsFs(), events.NoopPublisher{}).GetCertificate, events.NewBus())

	t.Cleanup(func() {
		assert.NoError(t, instance.Close())
	})

	return instance
}

func target(port int, handler http.Handler) server.Target {
	return server.Target{Address: hosts.Loopback.Port(port).String(), Handler: handler}
}

func TestRestartKeepsUnchangedListeners(t *testing.T) {
	t.Run("connections opened before the restart are served by the new handler", func(t *testing.T) {
		port := testutils.GetFreePort(t)
		instance := newInstance(t)
		client := newClient()
		url := hosts.Loopback.HTTPPort(port).String()

		require.NoError(t, instance.Start(t.Context(), []server.Target{target(port, respond("first"))}))

		body, err := get(t, client, url)
		require.NoError(t, err)
		require.Equal(t, "first", body)

		conn, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", hosts.Loopback.Port(port).String())
		require.NoError(t, err)

		defer conn.Close()

		retired, err := instance.Restart(t.Context(), []server.Target{target(port, respond("second"))})
		require.NoError(t, err)

		body, err = get(t, client, url)
		require.NoError(t, err)
		assert.Equal(t, "second", body)

		_, err = io.WriteString(conn, "GET / HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: close\r\n\r\n")
		require.NoError(t, err)

		response, err := http.ReadResponse(bufio.NewReader(conn), nil)
		require.NoError(t, err)

		defer response.Body.Close()

		raw, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		assert.Equal(t, "second", string(raw))

		testutils.WaitClosed(t, retired)
	})

	t.Run("an in-flight request finishes on the previous handler", func(t *testing.T) {
		port := testutils.GetFreePort(t)
		instance := newInstance(t)
		url := hosts.Loopback.HTTPPort(port).String()
		slow := newBlocking()

		require.NoError(t, instance.Start(t.Context(), []server.Target{target(port, slow)}))

		result := make(chan string, 1)

		go func() {
			body, err := get(t, newClient(), url)
			assert.NoError(t, err)

			result <- body
		}()

		<-slow.entered

		retired, err := instance.Restart(t.Context(), []server.Target{target(port, respond("second"))})
		require.NoError(t, err)

		body, err := get(t, newClient(), url)
		require.NoError(t, err)
		assert.Equal(t, "second", body)

		select {
		case <-retired:
			t.Fatal("the previous targets were retired while a request was in flight")
		default:
		}

		close(slow.release)

		assert.Equal(t, "slow", <-result)
		testutils.WaitClosed(t, retired)
	})

	t.Run("a removed port stops accepting while its request finishes", func(t *testing.T) {
		ports := testutils.GetFreePorts(t, 2)
		instance := newInstance(t)
		slow := newBlocking()

		require.NoError(t, instance.Start(t.Context(), []server.Target{target(ports[0], slow)}))

		result := make(chan string, 1)

		go func() {
			body, err := get(t, newClient(), hosts.Loopback.HTTPPort(ports[0]).String())
			assert.NoError(t, err)

			result <- body
		}()

		<-slow.entered

		retired, err := instance.Restart(t.Context(), []server.Target{target(ports[1], respond("second"))})
		require.NoError(t, err)

		assert.True(t, testutils.IsPortFree(ports[0]))

		body, err := get(t, newClient(), hosts.Loopback.HTTPPort(ports[1]).String())
		require.NoError(t, err)
		assert.Equal(t, "second", body)

		close(slow.release)

		assert.Equal(t, "slow", <-result)
		testutils.WaitClosed(t, retired)
	})

	t.Run("shutdown does not wait for the previous targets to drain", func(t *testing.T) {
		port := testutils.GetFreePort(t)
		instance := newInstance(t)
		slow := newBlocking()
		slow.stubborn = true

		defer close(slow.release)

		server.SetShutdownTimeout(instance, time.Minute)

		require.NoError(t, instance.Start(t.Context(), []server.Target{target(port, slow)}))

		go func() {
			_, _ = get(t, newClient(), hosts.Loopback.HTTPPort(port).String())
		}()

		<-slow.entered

		retired, err := instance.Restart(t.Context(), []server.Target{target(port, respond("second"))})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		require.Error(t, instance.Shutdown(ctx))
		testutils.WaitClosed(t, retired)
	})
}

func TestRestartSwitchesSchemeOnTheSamePort(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	fs := afero.NewOsFs()

	certPath, keyPath, err := certs.Generate(fs, filepath.Join(fakeHome, ".config", "uncors"), 10)
	require.NoError(t, err)

	caCert, _, err := certs.Load(fs, certPath, keyPath)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	manager := certs.NewManager(fs, events.NoopPublisher{})
	manager.SetHosts([]string{"127.0.0.1"})

	instance := server.New(manager.GetCertificate, events.NewBus())

	defer func() {
		assert.NoError(t, instance.Close())
	}()

	port := testutils.GetFreePort(t)
	plain := target(port, respond("plain"))
	secure := target(port, respond("secure"))
	secure.EnableTLS = true

	httpsClient := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool, ServerName: "127.0.0.1"},
	}}
	httpURL := hosts.Loopback.HTTPPort(port).String()
	httpsURL := hosts.Loopback.HTTPSPort(port).String()

	require.NoError(t, instance.Start(t.Context(), []server.Target{plain}))

	body, err := get(t, newClient(), httpURL)
	require.NoError(t, err)
	assert.Equal(t, "plain", body)

	_, err = instance.Restart(t.Context(), []server.Target{secure})
	require.NoError(t, err)

	body, err = get(t, httpsClient, httpsURL)
	require.NoError(t, err)
	assert.Equal(t, "secure", body)

	body, err = get(t, newClient(), httpURL)
	require.NoError(t, err)
	assert.Contains(t, body, "HTTP request to an HTTPS server")

	_, err = instance.Restart(t.Context(), []server.Target{plain})
	require.NoError(t, err)

	body, err = get(t, newClient(), httpURL)
	require.NoError(t, err)
	assert.Equal(t, "plain", body)
}
