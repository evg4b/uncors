package uncors_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
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

func TestHandlerWithHTTP(t *testing.T) {
	container := di.NewContainer(afero.NewMemMapFs(), io.Discard)
	app := uncors.CreateUncors(container.Fs(), container.Server(), container.CliOutput(), "test")

	targetServer := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello from target: %s %s", r.Method, r.URL.Path) //nolint:gosec // G705: test handler
	}))
	defer targetServer.Close()

	port := testutils.GetFreePort(t)

	err := app.Start(t.Context(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   hosts.Parse(targetServer.URL),
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
			url := testutils.JoinPath(hosts.Loopback.HTTPPort(port).String(), "api", method)
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
			testutils.JoinPath(hosts.Loopback.HTTPPort(port).String(), "/api/OPTIONS"),
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
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	fs := afero.NewOsFs()
	require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

	container := di.NewContainer(fs, io.Discard)
	app := uncors.CreateUncors(container.Fs(), container.Server(), container.CliOutput(), "test")

	targetServer := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "HTTPS response: %s %s", r.Method, r.URL.Path) //nolint:gosec // G705: test handler
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

	err = app.Start(t.Context(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPSPort(port),
				To:   hosts.Parse(targetServer.URL),
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
			url := testutils.JoinPath(hosts.Loopback.HTTPSPort(port).String(), "secure", method)
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
			testutils.JoinPath(hosts.Loopback.HTTPSPort(port).String(), "/secure/OPTIONS"),
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
	container := di.NewContainer(afero.NewMemMapFs(), io.Discard)
	app := uncors.CreateUncors(container.Fs(), container.Server(), container.CliOutput(), "test")

	mockFile := "/mock-response.json"
	mockContent := `{"message":"mocked"}`
	require.NoError(t, afero.WriteFile(container.Fs(), mockFile, []byte(mockContent), 0o644))

	port := testutils.GetFreePort(t)

	cfg := &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   hosts.Parse("http://example.com"),
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

	url := testutils.JoinPath(hosts.Loopback.HTTPPort(port).String(), "api", "mock")
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
	container := di.NewContainer(fs, io.Discard)
	app := uncors.CreateUncors(container.Fs(), container.Server(), container.CliOutput(), "test")

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
				To:   hosts.Parse("http://example.com"),
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
		url := testutils.JoinPath(hosts.Loopback.HTTPPort(port).String(), "static", "/")
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
		url := testutils.JoinPath(hosts.Loopback.HTTPPort(port).String(), "static", "test.txt")
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
	container := di.NewContainer(afero.NewMemMapFs(), io.Discard)
	app := uncors.CreateUncors(container.Fs(), container.Server(), container.CliOutput(), "test")

	callCount := 0

	targetServer := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
				To:   hosts.Parse(targetServer.URL),
				Cache: config.CacheGlobs{
					"/cached/*",
				},
			},
		},
		CacheConfig: config.CacheConfig{
			Methods:        []string{http.MethodGet},
			ExpirationTime: time.Minute,
			MaxSize:        100 * 1024 * 1024,
		},
	}

	require.NoError(t, app.Start(t.Context(), cfg))
	defer app.Close()

	client := http.DefaultClient
	baseURL := hosts.Loopback.HTTPPort(port).String()

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
	container := di.NewContainer(afero.NewMemMapFs(), io.Discard)
	app := uncors.CreateUncors(container.Fs(), container.Server(), container.CliOutput(), "test")

	server1 := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "Server 1")
	}))
	defer server1.Close()

	server2 := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "Server 2")
	}))
	defer server2.Close()

	port1 := testutils.GetFreePort(t)
	port2 := testutils.GetFreePort(t)

	cfg := &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port1),
				To:   hosts.Parse(server1.URL),
			},
			{
				From: hosts.Loopback.HTTPPort(port2),
				To:   hosts.Parse(server2.URL),
			},
		},
	}

	require.NoError(t, app.Start(t.Context(), cfg))
	defer app.Close()

	client := http.DefaultClient

	t.Run("mapping 1", func(t *testing.T) {
		url := hosts.Loopback.HTTPPort(port1).String()
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
		url := hosts.Loopback.HTTPPort(port2).String()
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
	container := di.NewContainer(afero.NewMemMapFs(), io.Discard)
	app := uncors.CreateUncors(container.Fs(), container.Server(), container.CliOutput(), "test")

	targetServer := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Path: %s, Host: %s", r.URL.Path, r.Host) //nolint:gosec // G705: test handler
	}))
	defer targetServer.Close()

	port := testutils.GetFreePort(t)

	cfg := &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   hosts.Parse(targetServer.URL),
				Rewrites: []config.RewritingOption{
					{
						From: targetServer.URL,
						To:   hosts.Loopback.HTTPPort(port).String(),
					},
				},
			},
		},
	}

	require.NoError(t, app.Start(t.Context(), cfg))
	defer app.Close()

	client := http.DefaultClient
	url := hosts.Loopback.HTTPPort(port).String() + "/test"

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

func TestHandlerWithRewritePath(t *testing.T) {
	container := di.NewContainer(afero.NewMemMapFs(), io.Discard)
	app := uncors.CreateUncors(container.Fs(), container.Server(), container.CliOutput(), "test")

	targetServer := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Path: %s", r.URL.Path) //nolint:gosec // G705: test handler
	}))
	defer targetServer.Close()

	port := testutils.GetFreePort(t)

	cfg := &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   hosts.Parse(targetServer.URL),
				Rewrites: []config.RewritingOption{
					{
						From: "/api/v1",
						To:   "/api/v2",
					},
				},
			},
		},
	}

	require.NoError(t, app.Start(t.Context(), cfg))
	defer app.Close()

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		hosts.Loopback.HTTPPort(port).String()+"/api/v1",
		nil,
	)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "/api/v2")
}

func TestHandlerWithOptions(t *testing.T) {
	container := di.NewContainer(afero.NewMemMapFs(), io.Discard)
	app := uncors.CreateUncors(container.Fs(), container.Server(), container.CliOutput(), "test")

	targetServer := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
				To:   hosts.Parse(targetServer.URL),
				OptionsHandling: config.OptionsHandling{
					Code:    http.StatusNoContent,
					Headers: customHeaders,
				},
			},
		},
	}

	require.NoError(t, app.Start(t.Context(), cfg))
	defer app.Close()

	url := hosts.Loopback.HTTPPort(port).String() + "/test"
	client := &http.Client{}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodOptions, url, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "custom-value", resp.Header.Get("X-Custom-Header"))
}

func TestHandlerWithScript(t *testing.T) {
	container := di.NewContainer(afero.NewMemMapFs(), io.Discard)
	app := uncors.CreateUncors(container.Fs(), container.Server(), container.CliOutput(), "test")

	port := testutils.GetFreePort(t)

	cfg := &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   hosts.Parse("http://example.com"),
				Scripts: config.Scripts{
					{
						Matcher: config.RequestMatcher{
							Path: "/script",
						},
						Script: `response:WriteHeader(201)`,
					},
				},
			},
		},
	}

	require.NoError(t, app.Start(t.Context(), cfg))
	defer app.Close()

	reqURL := hosts.Loopback.HTTPPort(port).String() + "/script"
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, reqURL, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}
