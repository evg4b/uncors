package uncors_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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

// Helper functions

func setupTestApp(t *testing.T) (*uncors.Uncors, afero.Fs) {
	t.Helper()

	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "test")

	return app, fs
}

func setupTLSClient(t *testing.T, fs afero.Fs) *http.Client {
	t.Helper()

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

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
				RootCAs:    pool,
				ServerName: "127.0.0.1",
			},
		},
	}
}

func getFreePort(t *testing.T) int {
	t.Helper()

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	return port
}

func doGetRequest(ctx context.Context, t *testing.T, url string) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	return resp, body
}

func doGetRequestWithClient(
	ctx context.Context,
	t *testing.T,
	client *http.Client,
	url string,
) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	return resp, body
}

// Tests

func TestCreateUncors(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := log.Default()
	version := "1.0.0"

	app := uncors.CreateUncors(fs, logger, version)

	assert.NotNil(t, app)
}

func TestUncorsApp(t *testing.T) {
	app, fs := setupTestApp(t)

	testResponceHeader := "# Test resrver"
	hostFmt := func(host string) string {
		return fmt.Sprintf("\tHost: %v", host)
	}
	methodFmt := func(method string) string {
		return fmt.Sprintf("\tMethod: %v", method)
	}
	urlFmt := func(method string) string {
		return fmt.Sprintf("\tURL: %v", method)
	}

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, testResponceHeader)
		fmt.Fprintln(w, methodFmt(r.Method))
		fmt.Fprintln(w, urlFmt(r.URL.String()))
		fmt.Fprintln(w, hostFmt(r.Host))
	}))
	defer targetServer.Close()

	client := setupTLSClient(t, fs)
	port := getFreePort(t)

	err := app.Start(t.Context(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   targetServer.URL,
			},
		},
	})
	require.NoError(t, err)

	defer func() {
		require.NoError(t, app.Close())
	}()

	t.Run("proxy", func(t *testing.T) {
		resp, bodyData := doGetRequestWithClient(t.Context(), t, client, hosts.Loopback.HTTPPort(port))

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		uri, err := url.Parse(targetServer.URL)
		require.NoError(t, err)

		assert.Contains(t, string(bodyData), uri.Host)
		assert.Contains(t, string(bodyData), methodFmt(http.MethodGet))
	})
}

func TestUncorsStart(t *testing.T) {
	app, _ := setupTestApp(t)

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}))
	defer targetServer.Close()

	port := getFreePort(t)

	err := app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   targetServer.URL,
			},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	resp, _ := doGetRequest(context.Background(), t, hosts.Loopback.HTTPPort(port))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUncorsRestart(t *testing.T) {
	app, _ := setupTestApp(t)

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "Server 1")
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "Server 2")
	}))
	defer server2.Close()

	port := getFreePort(t)

	err := app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   server1.URL,
			},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	_, body := doGetRequest(context.Background(), t, hosts.Loopback.HTTPPort(port))
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

	_, body = doGetRequest(context.Background(), t, hosts.Loopback.HTTPPort(port))
	assert.Equal(t, "Server 2", string(body))
}

func TestUncorsClose(t *testing.T) {
	app, _ := setupTestApp(t)

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	port := getFreePort(t)

	err := app.Start(context.Background(), &config.UncorsConfig{
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
	app, _ := setupTestApp(t)

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	port := getFreePort(t)

	err := app.Start(context.Background(), &config.UncorsConfig{
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
	app, _ := setupTestApp(t)

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	port := getFreePort(t)

	err := app.Start(context.Background(), &config.UncorsConfig{
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
	app, fs := setupTestApp(t)

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "HTTPS OK")
	}))
	defer targetServer.Close()

	client := setupTLSClient(t, fs)
	port := getFreePort(t)

	err := app.Start(context.Background(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPSPort(port),
				To:   targetServer.URL,
			},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	resp, body := doGetRequestWithClient(context.Background(), t, client, hosts.Loopback.HTTPSPort(port))

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "HTTPS OK", string(body))
}

func TestUncorsWithMixedHTTPAndHTTPS(t *testing.T) {
	app, fs := setupTestApp(t)

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "HTTP")
	}))
	defer httpServer.Close()

	httpsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "HTTPS")
	}))
	defer httpsServer.Close()

	tlsClient := setupTLSClient(t, fs)
	httpPort := getFreePort(t)
	httpsPort := getFreePort(t)

	err := app.Start(context.Background(), &config.UncorsConfig{
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
		_, body := doGetRequest(context.Background(), t, hosts.Loopback.HTTPPort(httpPort))
		assert.Equal(t, "HTTP", string(body))
	})

	t.Run("HTTPS endpoint", func(t *testing.T) {
		_, body := doGetRequestWithClient(context.Background(), t, tlsClient, hosts.Loopback.HTTPSPort(httpsPort))
		assert.Equal(t, "HTTPS", string(body))
	})
}

func TestUncorsWithComplexConfiguration(t *testing.T) {
	app, fs := setupTestApp(t)

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

	port := getFreePort(t)

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
		_, body := doGetRequest(context.Background(), t, hosts.Loopback.HTTPPort(port)+"/static/")
		assert.Contains(t, string(body), "Static")
	})

	t.Run("mock endpoint", func(t *testing.T) {
		_, body := doGetRequest(context.Background(), t, hosts.Loopback.HTTPPort(port)+"/api/mock")
		assert.JSONEq(t, `{"mocked":true}`, string(body))
	})

	t.Run("proxied content", func(t *testing.T) {
		_, body := doGetRequest(context.Background(), t, hosts.Loopback.HTTPPort(port)+"/other")
		assert.Equal(t, "Proxied", string(body))
	})
}
