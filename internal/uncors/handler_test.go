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
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerWithHTTP(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.New(io.Discard), "test")

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello from target: %s %s", r.Method, r.URL.Path)
	}))
	defer targetServer.Close()

	port := testutils.GetFreePort(t)

	err := app.Start(t.Context(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   targetServer.URL,
			},
		},
	})
	require.NoError(t, err)

	defer app.Close()

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			url := testutils.JoinPath(hosts.Loopback.HTTPPort(port), "api", method)
			req, err := http.NewRequestWithContext(t.Context(), method, url, nil)
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "test-value", resp.Header.Get("X-Test-Header"))

			if method != http.MethodHead {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(body), fmt.Sprintf("Hello from target: %s /api/%s", method, method))
			}
		})
	}

	t.Run("OPTIONS request", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodOptions,
			testutils.JoinPath(hosts.Loopback.HTTPPort(port), "/api/OPTIONS"),
			nil,
		)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Access-Control-Allow-Origin"), "*")
	})
}

func TestHandlerWithHTTPS(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.New(io.Discard), "test")

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "HTTPS response: %s %s", r.Method, r.URL.Path)
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
				MinVersion: tls.VersionTLS13,
				RootCAs:    pool,
				ServerName: "127.0.0.1",
			},
		},
	}

	port := testutils.GetFreePort(t)

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

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			url := testutils.JoinPath(hosts.Loopback.HTTPSPort(port), "secure", method)
			req, err := http.NewRequestWithContext(t.Context(), method, url, nil)
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)

			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "test-value", resp.Header.Get("X-Test-Header"))

			if method != http.MethodHead {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(body), fmt.Sprintf("HTTPS response: %s /secure/%s", method, method))
			}
		})
	}

	t.Run("OPTIONS request", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodOptions,
			testutils.JoinPath(hosts.Loopback.HTTPSPort(port), "/secure/OPTIONS"),
			nil,
		)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Access-Control-Allow-Origin"), "*")
	})
}

func TestHandlerWithMockMiddleware(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.New(io.Discard), "test")

	mockFile := "/mock-response.json"
	mockContent := `{"message":"mocked"}`
	require.NoError(t, afero.WriteFile(fs, mockFile, []byte(mockContent), 0o644))

	port := testutils.GetFreePort(t)

	cfg := &config.UncorsConfig{
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
							Code: http.StatusOK,
							File: mockFile,
						},
					},
				},
			},
		},
	}

	require.NoError(t, app.Start(t.Context(), cfg))
	defer app.Close()

	url := testutils.JoinPath(hosts.Loopback.HTTPPort(port), "api", "mock")
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, mockContent, string(body))
}

func TestHandlerWithStaticMiddleware(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.New(io.Discard), "test")

	staticDir := "/static"
	indexFile := filepath.Join(staticDir, "index.html")
	textFile := filepath.Join(staticDir, "test.txt")

	require.NoError(t, fs.MkdirAll(staticDir, 0o755))
	require.NoError(t, afero.WriteFile(fs, indexFile, []byte("<html>Static Content</html>"), 0o644))
	require.NoError(t, afero.WriteFile(fs, textFile, []byte("test file content"), 0o644))

	port := testutils.GetFreePort(t)

	cfg := &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   "http://example.com",
				Statics: []config.StaticDirectory{
					{
						Path:  staticDir,
						Dir:   staticDir,
						Index: "index.html",
					},
				},
			},
		},
	}

	require.NoError(t, app.Start(t.Context(), cfg))
	defer app.Close()

	t.Run("serve index file", func(t *testing.T) {
		url := testutils.JoinPath(hosts.Loopback.HTTPPort(port), "static", "/")
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), "Static Content")
	})

	t.Run("serve specific file", func(t *testing.T) {
		url := testutils.JoinPath(hosts.Loopback.HTTPPort(port), "static", "test.txt")
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "test file content", string(body))
	})
}

func TestHandlerWithCache(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.New(io.Discard), "test")

	callCount := 0

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Response #%d", callCount)
	}))
	defer targetServer.Close()

	port := testutils.GetFreePort(t)

	cfg := &config.UncorsConfig{
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
			Methods:        []string{http.MethodGet},
			ExpirationTime: time.Minute,
			ClearTime:      2 * time.Minute,
		},
	}

	require.NoError(t, app.Start(t.Context(), cfg))
	defer app.Close()

	client := http.DefaultClient
	baseURL := hosts.Loopback.HTTPPort(port)

	t.Run("first request", func(t *testing.T) {
		url := testutils.JoinPath(baseURL, "cached", "test")
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), "Response #1")
	})

	t.Run("cached request", func(t *testing.T) {
		url := testutils.JoinPath(baseURL, "cached", "test")
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), "Response #1")
		assert.Equal(t, 1, callCount, "should use cached response")
	})

	t.Run("non-cached path", func(t *testing.T) {
		url := testutils.JoinPath(baseURL, "other", "path")
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), "Response #2")
		assert.Equal(t, 2, callCount, "should not use cache for different path")
	})
}

func TestHandlerWithMultipleMappings(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.New(io.Discard), "test")

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "Server 1")
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "Server 2")
	}))
	defer server2.Close()

	port1 := testutils.GetFreePort(t)
	port2 := testutils.GetFreePort(t)

	cfg := &config.UncorsConfig{
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
	}

	require.NoError(t, app.Start(t.Context(), cfg))
	defer app.Close()

	client := http.DefaultClient

	t.Run("mapping 1", func(t *testing.T) {
		url := hosts.Loopback.HTTPPort(port1)
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "Server 1", string(body))
	})

	t.Run("mapping 2", func(t *testing.T) {
		url := hosts.Loopback.HTTPPort(port2)
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "Server 2", string(body))
	})
}

func TestHandlerWithRewrite(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.New(io.Discard), "test")

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Path: %s, Host: %s", r.URL.Path, r.Host)
	}))
	defer targetServer.Close()

	port := testutils.GetFreePort(t)

	cfg := &config.UncorsConfig{
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
	}

	require.NoError(t, app.Start(t.Context(), cfg))
	defer app.Close()

	client := http.DefaultClient
	url := hosts.Loopback.HTTPPort(port) + "/test"

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "/test")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandlerWithOptions(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.New(io.Discard), "test")

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	port := testutils.GetFreePort(t)

	customHeaders := map[string]string{
		"X-Custom-Header": "custom-value",
	}

	cfg := &config.UncorsConfig{
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
	}

	require.NoError(t, app.Start(t.Context(), cfg))
	defer app.Close()

	url := hosts.Loopback.HTTPPort(port) + "/test"
	client := &http.Client{}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodOptions, url, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "custom-value", resp.Header.Get("X-Custom-Header"))
}
