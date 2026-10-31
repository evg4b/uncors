//go:build integration

package proxy_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyUnreachableUpstream(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://broken.local"),
			To:   hosts.Loopback.HTTPPort(1),
		}},
	})

	req := integration.NewRequest(t, http.MethodGet, env.URL("broken.local", "/api/users"))
	req.Header.Set("Origin", "https://front.local")

	result := env.Do(t, req)

	t.Run("answers 502 with the failed upstream request in the body", func(t *testing.T) {
		assert.Equal(t, http.StatusBadGateway, result.Response.StatusCode)
		assert.Equal(t, "text/plain; charset=utf-8", result.Response.Header.Get("Content-Type"))
		assert.Contains(t, result.BodyString(),
			`An error occurred: failed to execute request: Get "http://127.0.0.1:1/api/users"`)
		assert.Contains(t, result.BodyString(), "connection refused")
	})

	t.Run("keeps CORS headers so the browser can read the error", func(t *testing.T) {
		assert.Equal(t, "https://front.local", result.Response.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "no-cache, no-store, max-age=0, must-revalidate", result.Response.Header.Get("Cache-Control"))
	})
}

func TestProxyMethods(t *testing.T) {
	type received struct {
		method string
		body   string
	}

	seen := make(chan received, 1)
	backend := integration.NewBackend(t, func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		assert.NoError(t, err)

		seen <- received{method: request.Method, body: string(body)}

		writer.WriteHeader(http.StatusAccepted)
		_, err = io.WriteString(writer, request.Method+" done")
		assert.NoError(t, err)
	})
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://methods.local"),
			To:   backend.AsHost(),
		}},
	})

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method+" reaches the backend with its body", func(t *testing.T) {
			payload := `{"method":"` + method + `"}`
			req := integration.NewRequest(t, method, env.URL("methods.local", "/items/1"))
			req.Body = io.NopCloser(strings.NewReader(payload))
			req.ContentLength = int64(len(payload))
			req.Header.Set("Content-Type", "application/json")

			result := env.Do(t, req)

			got := <-seen
			assert.Equal(t, method, got.method)
			assert.Equal(t, payload, got.body)
			assert.Contains(t, result.RawBackendRequest(t), "Content-Type: application/json\r\n")

			assert.Equal(t, http.StatusAccepted, result.Response.StatusCode)
			assert.Equal(t, method+" done", result.BodyString())
		})
	}

	t.Run("a large body arrives intact", func(t *testing.T) {
		payload := strings.Repeat("0123456789", 200_000)
		req := integration.NewRequest(t, http.MethodPost, env.URL("methods.local", "/upload"))
		req.Body = io.NopCloser(strings.NewReader(payload))
		req.ContentLength = int64(len(payload))

		result := env.Do(t, req)

		got := <-seen
		assert.Len(t, got.body, len(payload))
		assert.Equal(t, payload, got.body)
		assert.Equal(t, http.StatusAccepted, result.Response.StatusCode)
	})
}

func TestProxyRequestHopByHopHeaders(t *testing.T) {
	env, _ := newProxyEnv(t)

	req := integration.NewRequest(t, http.MethodGet, proxyURL(env, "/echo"))
	req.Header.Set("Connection", "X-Client-Hop")
	req.Header.Set("X-Client-Hop", "1")
	req.Header.Set("Proxy-Authorization", "Basic Zm9vOmJhcg==")
	req.Header.Set("X-Kept", "yes")

	dump := env.Do(t, req).RawBackendRequest(t)

	assert.NotContains(t, dump, "X-Client-Hop")
	assert.NotContains(t, dump, "Proxy-Authorization")
	assert.Contains(t, dump, "X-Kept: yes\r\n")
}

func TestProxyCompressedResponse(t *testing.T) {
	const text = "compressed payload"

	var compressed bytes.Buffer

	writer := gzip.NewWriter(&compressed)
	_, err := writer.Write([]byte(text))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	backend := integration.NewBackend(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Accept-Encoding") != "br, gzip" {
			_, err := io.WriteString(writer, text)
			assert.NoError(t, err)

			return
		}

		writer.Header().Set("Content-Encoding", "gzip")
		_, err := writer.Write(compressed.Bytes())
		assert.NoError(t, err)
	})
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://gzip.local"),
			To:   backend.AsHost(),
		}},
	})

	req := integration.NewRequest(t, http.MethodGet, env.URL("gzip.local", "/"))
	req.Header.Set("Accept-Encoding", "br, gzip")

	result := env.Do(t, req)

	assert.Contains(t, result.RawBackendRequest(t), "Accept-Encoding: br, gzip\r\n")
	assert.Equal(t, "gzip", result.Response.Header.Get("Content-Encoding"))
	assert.Equal(t, compressed.Bytes(), []byte(result.BodyString()), "the body must pass through still compressed")
}

func TestProxyHTTPSUpstream(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)

	to, err := urlt.ParseHost(upstream.URL)
	require.NoError(t, err)

	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{From: hosts.Parse("https://secure.local"), To: *to}},
	})

	result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("secure.local", "/api")))

	t.Run("speaks TLS to the upstream and refuses a certificate the system does not trust", func(t *testing.T) {
		assert.Equal(t, http.StatusBadGateway, result.Response.StatusCode)
		assert.Contains(t, result.BodyString(), `Get "`+upstream.URL+`/api"`)
		assert.Contains(t, result.BodyString(), "x509: certificate signed by unknown authority")
	})
}

func TestProxyUpstreamHandshakeTimeout(t *testing.T) {
	port := testutils.GetFreePort(t)

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp4", hosts.Loopback.Port(port).String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		var held []net.Conn

		for {
			conn, err := listener.Accept()
			if err != nil {
				for _, conn := range held {
					_ = conn.Close()
				}

				return
			}

			held = append(held, conn)
		}
	}()

	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://stalled.local"),
			To:   hosts.Loopback.HTTPSPort(port),
		}},
	})

	result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("stalled.local", "/")))

	assert.Equal(t, http.StatusGatewayTimeout, result.Response.StatusCode)
	assert.Contains(t, result.BodyString(), "TLS handshake timeout")
}

func TestProxySchemelessTarget(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	to := backend.AsHost()
	to.Scheme = ""

	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{
			{From: hosts.Parse("http://plain.local"), To: to},
			{From: hosts.Parse("https://secure.local"), To: to},
		},
	})

	t.Run("an http source calls the upstream over http", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("plain.local", "/echo")))

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Contains(t, result.BackendRequest(t), "GET /echo HTTP/1.1")
	})

	t.Run("an https source calls the upstream over https", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("secure.local", "/echo")))

		assert.Equal(t, http.StatusBadGateway, result.Response.StatusCode)
		assert.Contains(t, result.BodyString(), `Get "https://`+to.String()+`/echo"`)
		assert.False(t, result.HasBackendRequest())
	})
}

func TestProxyThroughUpstreamProxy(t *testing.T) {
	forward := integration.NewBackend(t, func(writer http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(writer, "via proxy")
		assert.NoError(t, err)
	})
	env := integration.New(t, forward, &config.UncorsConfig{
		Proxy: forward.URL(),
		Mappings: config.Mappings{{
			From: hosts.Parse("https://app.local"),
			To:   hosts.Parse("http://api.upstream.test"),
		}},
	})

	req := integration.NewRequest(t, http.MethodGet, env.RawURL("app.local", "/users?page=2"))
	req.Header.Set("Origin", "https://app.local")

	result := env.Do(t, req)

	assert.Equal(t, "via proxy", result.BodyString())
	assert.Equal(t, "https://app.local", result.Response.Header.Get("Access-Control-Allow-Origin"))

	dump := result.BackendRequest(t)
	assert.True(t, strings.HasPrefix(dump, "GET http://api.upstream.test/users?page=2 HTTP/1.1\n"), dump)
	assert.Contains(t, dump, "Origin: http://api.upstream.test\n")
}
