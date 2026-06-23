package uncors_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const version = "1.0.0"

func TestCreateUncors(t *testing.T) {
	container := di.NewContainer(di.WithVersion(version))
	defer testutils.Close(t, container)

	app := uncors.CreateUncors(container)

	assert.NotNil(t, app)
}

func TestUncorsApp(t *testing.T) {
	container := di.NewContainer(di.WithVersion(version))
	defer testutils.Close(t, container)

	app := uncors.CreateUncors(container)
	fs := container.Fs()

	testResponceHeader := "# Test resrver"
	hostFmt := func(host string) string { return fmt.Sprintf("\tHost: %v", host) }
	methodFmt := func(method string) string { return fmt.Sprintf("\tMethod: %v", method) }
	urlFmt := func(method string) string { return fmt.Sprintf("\tURL: %v", method) }

	targetServer := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, testResponceHeader)
		fmt.Fprintln(w, methodFmt(r.Method))
		fmt.Fprintln(w, urlFmt(r.URL.String()))
		fmt.Fprintln(w, hostFmt(r.Host))
	}))
	defer targetServer.Close()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	certPath, keyPath, err := server.GenerateCA(server.CAConfig{
		Fs:           fs,
		ValidityDays: 10,
		OutputDir:    filepath.Join(homeDir, ".config", "uncors"),
	})
	require.NoError(t, err)

	caCert, _, err := server.LoadCA(fs, certPath, keyPath)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
				RootCAs:    pool,
				ServerName: "127.0.0.1",
			},
		},
	}

	port := testutils.GetFreePort(t)

	err = app.Start(t.Context(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{From: hosts.Loopback.HTTPPort(port), To: hosts.Parse(targetServer.URL)},
		},
	})
	require.NoError(t, err)

	defer func() { require.NoError(t, app.Close()) }()

	t.Run("proxy", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			hosts.Loopback.HTTPPort(port).String(),
			nil,
		)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)

		bodyData, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		uri, err := url.Parse(targetServer.URL)
		require.NoError(t, err)

		assert.Contains(t, string(bodyData), uri.Host)
		assert.Contains(t, string(bodyData), methodFmt(http.MethodGet))
	})
}

func TestUncorsStart(t *testing.T) {
	container := di.NewContainer(di.WithVersion(version))
	defer testutils.Close(t, container)

	app := uncors.CreateUncors(container)

	targetServer := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}))
	defer targetServer.Close()

	port := testutils.GetFreePort(t)

	err := app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{From: hosts.Loopback.HTTPPort(port), To: hosts.Parse(targetServer.URL)},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		hosts.Loopback.HTTPPort(port).String(),
		nil,
	)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "OK", string(body))
}

func TestUncorsRestart(t *testing.T) {
	container := di.NewContainer(di.WithVersion(version))
	defer testutils.Close(t, container)

	app := uncors.CreateUncors(container)

	server1 := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "Server 1")
	}))
	defer server1.Close()

	server2 := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "Server 2")
	}))
	defer server2.Close()

	port := testutils.GetFreePort(t)

	err := app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{From: hosts.Loopback.HTTPPort(port), To: hosts.Parse(server1.URL)},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	req1, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		hosts.Loopback.HTTPPort(port).String(),
		nil,
	)
	require.NoError(t, err)
	resp1, err := http.DefaultClient.Do(req1)
	require.NoError(t, err)
	body1, err := io.ReadAll(resp1.Body)
	require.NoError(t, err)
	resp1.Body.Close()
	assert.Equal(t, "Server 1", string(body1))

	err = app.Restart(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{From: hosts.Loopback.HTTPPort(port), To: hosts.Parse(server2.URL)},
		},
	})
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	req2, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		hosts.Loopback.HTTPPort(port).String(),
		nil,
	)
	require.NoError(t, err)
	resp2, err := http.DefaultClient.Do(req2)
	require.NoError(t, err)
	body2, err := io.ReadAll(resp2.Body)
	require.NoError(t, err)
	resp2.Body.Close()
	assert.Equal(t, "Server 2", string(body2))
}

func TestUncorsClose(t *testing.T) {
	container := di.NewContainer(di.WithVersion(version))
	defer testutils.Close(t, container)

	app := uncors.CreateUncors(container)

	targetServer := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	port := testutils.GetFreePort(t)
	err := app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{From: hosts.Loopback.HTTPPort(port), To: hosts.Parse(targetServer.URL)},
		},
	})
	require.NoError(t, err)

	err = app.Close()
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		hosts.Loopback.HTTPPort(port).String(),
		nil,
	)
	require.NoError(t, err)
	_, err = http.DefaultClient.Do(req)
	assert.Error(t, err)
}

func TestUncorsShutdown(t *testing.T) {
	container := di.NewContainer(di.WithVersion(version))
	defer testutils.Close(t, container)

	app := uncors.CreateUncors(container)

	targetServer := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	port := testutils.GetFreePort(t)
	err := app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{From: hosts.Loopback.HTTPPort(port), To: hosts.Parse(targetServer.URL)},
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = app.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestUncorsWait(t *testing.T) {
	container := di.NewContainer(di.WithVersion(version))
	defer testutils.Close(t, container)

	app := uncors.CreateUncors(container)

	targetServer := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	port := testutils.GetFreePort(t)
	err := app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{From: hosts.Loopback.HTTPPort(port), To: hosts.Parse(targetServer.URL)},
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
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	fs := afero.NewOsFs()
	require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

	container := di.NewContainer(di.WithFs(fs))
	defer testutils.Close(t, container)

	app := uncors.CreateUncors(container)

	targetServer := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "HTTPS OK")
	}))
	defer targetServer.Close()

	caDir := filepath.Join(fakeHome, ".config", "uncors")
	certPath, keyPath, err := server.GenerateCA(server.CAConfig{
		Fs:           fs,
		ValidityDays: 10,
		OutputDir:    caDir,
	})
	require.NoError(t, err)
	caCert, _, err := server.LoadCA(fs, certPath, keyPath)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
				RootCAs:    pool,
				ServerName: "127.0.0.1",
			},
		},
	}

	port := testutils.GetFreePort(t)
	err = app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{From: hosts.Loopback.HTTPSPort(port), To: hosts.Parse(targetServer.URL)},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		hosts.Loopback.HTTPSPort(port).String(),
		nil,
	)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "HTTPS OK", string(body))
}

func TestUncorsWithMixedHTTPAndHTTPS(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	fs := afero.NewOsFs()
	require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

	container := di.NewContainer(di.WithFs(fs))
	defer testutils.Close(t, container)

	app := uncors.CreateUncors(container)

	httpServer := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "HTTP")
	}))
	defer httpServer.Close()

	httpsServer := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "HTTPS")
	}))
	defer httpsServer.Close()

	caDir := filepath.Join(fakeHome, ".config", "uncors")
	certPath, keyPath, err := server.GenerateCA(server.CAConfig{
		Fs: fs, ValidityDays: 10, OutputDir: caDir,
	})
	require.NoError(t, err)
	caCert, _, err := server.LoadCA(fs, certPath, keyPath)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	tlsClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
				RootCAs:    pool,
				ServerName: "127.0.0.1",
			},
		},
	}

	httpPort := testutils.GetFreePort(t)
	httpsPort := testutils.GetFreePort(t)

	err = app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{From: hosts.Loopback.HTTPPort(httpPort), To: hosts.Parse(httpServer.URL)},
			{From: hosts.Loopback.HTTPSPort(httpsPort), To: hosts.Parse(httpsServer.URL)},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	t.Run("HTTP endpoint", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			hosts.Loopback.HTTPPort(httpPort).String(),
			nil,
		)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, "HTTP", string(body))
	})

	t.Run("HTTPS endpoint", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			hosts.Loopback.HTTPSPort(httpsPort).String(),
			nil,
		)
		require.NoError(t, err)
		resp, err := tlsClient.Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, "HTTPS", string(body))
	})
}

func TestUncorsWithComplexConfiguration(t *testing.T) {
	container := di.NewContainer(di.WithVersion(version))
	defer testutils.Close(t, container)

	app := uncors.CreateUncors(container)
	fs := container.Fs()

	require.NoError(t, fs.MkdirAll("/static", 0o755))
	require.NoError(t, afero.WriteFile(fs, "/static/index.html", []byte("Static"), 0o644))
	require.NoError(t, afero.WriteFile(fs, "/mock.json", []byte(`{"mocked":true}`), 0o644))

	targetServer := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Proxied")
	}))
	defer targetServer.Close()

	port := testutils.GetFreePort(t)
	err := app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   hosts.Parse(targetServer.URL),
				Statics: []config.StaticDirectory{
					{Path: "/static", Dir: "/static", Index: "index.html"},
				},
				Mocks: []config.Mock{
					{
						Matcher: config.RequestMatcher{Path: "/api/mock"},
						Response: config.Response{
							Code: 200, File: "/mock.json",
						},
					},
				},
				Cache: config.CacheGlobs{"/cache/*"},
			},
		},
		CacheConfig: config.CacheConfig{
			Methods:        []string{"GET"},
			ExpirationTime: 1 * time.Minute,
			MaxSize:        100 * 1024 * 1024,
		},
	})
	require.NoError(t, err)

	defer app.Close()

	t.Run("static content", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			testutils.JoinPath(hosts.Loopback.HTTPPort(port).String(), "static"),
			nil,
		)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Contains(t, string(body), "Static")
	})

	t.Run("mock endpoint", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			testutils.JoinPath(hosts.Loopback.HTTPPort(port).String(), "api", "mock"),
			nil,
		)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()
		assert.JSONEq(t, `{"mocked":true}`, string(body))
	})

	t.Run("proxied content", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			testutils.JoinPath(hosts.Loopback.HTTPPort(port).String(), "other"),
			nil,
		)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, "Proxied", string(body))
	})
}
