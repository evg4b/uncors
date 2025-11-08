package uncors_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/config"
	infraTls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/phayes/freeport"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUncors(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := log.Default()
	version := "1.0.0"

	app := uncors.CreateUncors(fs, logger, version)

	assert.NotNil(t, app)
}

func TestUncorsStart(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "test")

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}))
	defer targetServer.Close()

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	err = app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   targetServer.URL,
			},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, hosts.Loopback.HTTPPort(port), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUncorsRestart(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "test")

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "Server 1")
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "Server 2")
	}))
	defer server2.Close()

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	err = app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   server1.URL,
			},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, hosts.Loopback.HTTPPort(port), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, "Server 1", string(body))

	err = app.Restart(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   server2.URL,
			},
		},
	})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, hosts.Loopback.HTTPPort(port), nil)
	require.NoError(t, err)

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)

	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, "Server 2", string(body))
}

func TestUncorsClose(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "test")

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	err = app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   targetServer.URL,
			},
		},
	})
	require.NoError(t, err)

	err = app.Close()
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, hosts.Loopback.HTTPPort(port), nil)
	require.NoError(t, err)

	_, err = http.DefaultClient.Do(req)
	assert.Error(t, err)
}

func TestUncorsShutdown(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "test")

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	err = app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   targetServer.URL,
			},
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = app.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestUncorsWait(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "test")

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	err = app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   targetServer.URL,
			},
		},
	})
	require.NoError(t, err)

	done := make(chan bool)

	go func() {
		app.Wait()

		done <- true
	}()

	go func() {
		time.Sleep(100 * time.Millisecond)
		app.Close()
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Wait() did not return in time")
	}
}

func TestUncorsWithHTTPSMapping(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "test")

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "HTTPS OK")
	}))
	defer targetServer.Close()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	certPath, keyPath, err := infraTls.GenerateCA(infraTls.CAConfig{
		Fs:           fs,
		ValidityDays: 10,
		OutputDir:    filepath.Join(homeDir, ".config", "uncors"),
	})
	require.NoError(t, err)

	caCert, _, err := infraTls.LoadCA(fs, certPath, keyPath)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS13,
				RootCAs:            pool,
				ServerName:         "127.0.0.1",
				InsecureSkipVerify: false,
			},
		},
	}

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	err = app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPSPort(port),
				To:   targetServer.URL,
			},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, hosts.Loopback.HTTPSPort(port), nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "HTTPS OK", string(body))
}

func TestUncorsWithMixedHTTPAndHTTPS(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "test")

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "HTTP")
	}))
	defer httpServer.Close()

	httpsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "HTTPS")
	}))
	defer httpsServer.Close()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	certPath, keyPath, err := infraTls.GenerateCA(infraTls.CAConfig{
		Fs:           fs,
		ValidityDays: 10,
		OutputDir:    filepath.Join(homeDir, ".config", "uncors"),
	})
	require.NoError(t, err)

	caCert, _, err := infraTls.LoadCA(fs, certPath, keyPath)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	tlsClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS13,
				RootCAs:            pool,
				ServerName:         "127.0.0.1",
				InsecureSkipVerify: false,
			},
		},
	}

	httpPort, err := freeport.GetFreePort()
	require.NoError(t, err)

	httpsPort, err := freeport.GetFreePort()
	require.NoError(t, err)

	err = app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(httpPort),
				To:   httpServer.URL,
			},
			{
				From: hosts.Loopback.HTTPSPort(httpsPort),
				To:   httpsServer.URL,
			},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	t.Run("HTTP endpoint", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, hosts.Loopback.HTTPPort(httpPort), nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "HTTP", string(body))
	})

	t.Run("HTTPS endpoint", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, hosts.Loopback.HTTPSPort(httpsPort), nil)
		require.NoError(t, err)

		resp, err := tlsClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "HTTPS", string(body))
	})
}

func TestUncorsWithComplexConfiguration(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "test")

	err := fs.MkdirAll("/static", 0o755)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "/static/index.html", []byte("Static"), 0o644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "/mock.json", []byte(`{"mocked":true}`), 0o644)
	require.NoError(t, err)

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Proxied")
	}))
	defer targetServer.Close()

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	err = app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   targetServer.URL,
				Statics: []config.StaticDirectory{
					{
						Path:  "/static",
						Dir:   "/static",
						Index: "index.html",
					},
				},
				Mocks: []config.Mock{
					{
						Matcher: config.RequestMatcher{
							Path: "/api/mock",
						},
						Response: config.Response{
							Code: 200,
							File: "/mock.json",
						},
					},
				},
				Cache: config.CacheGlobs{"/cache/*"},
			},
		},
		CacheConfig: config.CacheConfig{
			Methods:        []string{"GET"},
			ExpirationTime: 1 * time.Minute,
			ClearTime:      2 * time.Minute,
		},
	})
	require.NoError(t, err)

	defer app.Close()

	t.Run("static content", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			hosts.Loopback.HTTPPort(port)+"/static/",
			nil,
		)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Static")
	})

	t.Run("mock endpoint", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			hosts.Loopback.HTTPPort(port)+"/api/mock",
			nil,
		)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		assert.JSONEq(t, `{"mocked":true}`, string(body))
	})

	t.Run("proxied content", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			hosts.Loopback.HTTPPort(port)+"/other",
			nil,
		)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "Proxied", string(body))
	})
}
