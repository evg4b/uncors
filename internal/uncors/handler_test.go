package uncors_test

import (
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

func TestHandlerWithHTTP(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "test")

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello from target: %s %s", r.Method, r.URL.Path)
	}))
	defer targetServer.Close()

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	err = app.Start(t.Context(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   targetServer.URL,
			},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	t.Run("GET request", func(t *testing.T) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, hosts.Loopback.HTTPPort(port)+"/test", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Hello from target: GET /test")
	})

	t.Run("POST request", func(t *testing.T) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, hosts.Loopback.HTTPPort(port)+"/api", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "POST /api")
	})
}

func TestHandlerWithHTTPS(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "test")

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "HTTPS response: %s", r.URL.Path)
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

	err = app.Start(t.Context(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPSPort(port),
				To:   targetServer.URL,
			},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		hosts.Loopback.HTTPSPort(port)+"/secure",
		nil,
	)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "HTTPS response: /secure")
}

func TestHandlerWithMockMiddleware(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "test")

	err := afero.WriteFile(fs, "/mock-response.json", []byte(`{"message":"mocked"}`), 0o644)
	require.NoError(t, err)

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	err = app.Start(t.Context(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   "http://example.com",
				Mocks: []config.Mock{
					{
						Matcher: config.RequestMatcher{
							Path: "/api/mock",
						},
						Response: config.Response{
							Code: 200,
							File: "/mock-response.json",
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, hosts.Loopback.HTTPPort(port)+"/api/mock", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.JSONEq(t, `{"message":"mocked"}`, string(body))
}

func TestHandlerWithStaticMiddleware(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "test")

	err := fs.MkdirAll("/static", 0o755)
	require.NoError(t, err)

	err = afero.WriteFile(fs, "/static/index.html", []byte("<html>Static Content</html>"), 0o644)
	require.NoError(t, err)

	err = afero.WriteFile(fs, "/static/test.txt", []byte("test file content"), 0o644)
	require.NoError(t, err)

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	err = app.Start(t.Context(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   "http://example.com",
				Statics: []config.StaticDirectory{
					{
						Path:  "/static",
						Dir:   "/static",
						Index: "index.html",
					},
				},
			},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	t.Run("serve index file", func(t *testing.T) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, hosts.Loopback.HTTPPort(port)+"/static/", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Static Content")
	})

	t.Run("serve specific file", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			hosts.Loopback.HTTPPort(port)+"/static/test.txt",
			nil,
		)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "test file content", string(body))
	})
}

func TestHandlerWithCache(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "test")

	callCount := 0

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Response #%d", callCount)
	}))
	defer targetServer.Close()

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	err = app.Start(t.Context(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   targetServer.URL,
				Cache: config.CacheGlobs{
					"/cached/*",
				},
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

	t.Run("first request", func(t *testing.T) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, hosts.Loopback.HTTPPort(port)+"/cached/test", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Response #1")
	})

	t.Run("cached request", func(t *testing.T) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, hosts.Loopback.HTTPPort(port)+"/cached/test", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Response #1")
		assert.Equal(t, 1, callCount, "should use cached response")
	})

	t.Run("non-cached path", func(t *testing.T) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, hosts.Loopback.HTTPPort(port)+"/other/path", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Response #2")
		assert.Equal(t, 2, callCount, "should not use cache for different path")
	})
}

func TestHandlerWithMultipleMappings(t *testing.T) {
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

	port1, err := freeport.GetFreePort()
	require.NoError(t, err)

	port2, err := freeport.GetFreePort()
	require.NoError(t, err)

	err = app.Start(t.Context(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port1),
				To:   server1.URL,
			},
			{
				From: hosts.Loopback.HTTPPort(port2),
				To:   server2.URL,
			},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	t.Run("mapping 1", func(t *testing.T) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, hosts.Loopback.HTTPPort(port1), nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "Server 1", string(body))
	})

	t.Run("mapping 2", func(t *testing.T) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, hosts.Loopback.HTTPPort(port2), nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "Server 2", string(body))
	})
}

func TestHandlerWithRewrite(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "test")

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Path: %s, Host: %s", r.URL.Path, r.Host)
	}))
	defer targetServer.Close()

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	err = app.Start(t.Context(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   targetServer.URL,
				Rewrites: []config.RewritingOption{
					{
						From: targetServer.URL,
						To:   hosts.Loopback.HTTPPort(port),
					},
				},
			},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, hosts.Loopback.HTTPPort(port)+"/test", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "/test")
}

func TestHandlerWithOptions(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "test")

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	customHeaders := map[string]string{
		"X-Custom-Header": "custom-value",
	}

	err = app.Start(t.Context(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   targetServer.URL,
				OptionsHandling: config.OptionsHandling{
					Code:    http.StatusNoContent,
					Headers: customHeaders,
				},
			},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodOptions, hosts.Loopback.HTTPPort(port)+"/test", nil)
	require.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "custom-value", resp.Header.Get("X-Custom-Header"))
}
