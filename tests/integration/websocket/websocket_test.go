//go:build integration

package websocket_test

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/websocket"
)

const (
	wait     = 5 * time.Second
	pollTick = 10 * time.Millisecond
)

type echoServer struct {
	*integration.Backend

	mu      sync.Mutex
	origins []string
	closed  map[string]int
}

func newEchoServer(t *testing.T) *echoServer {
	t.Helper()

	echo := &echoServer{closed: map[string]int{}}

	handler := websocket.Handler(func(conn *websocket.Conn) {
		request := conn.Request()
		echo.opened(request)

		for {
			var message string

			err := websocket.Message.Receive(conn, &message)
			if err != nil {
				echo.markClosed(request.URL.Path)

				return
			}

			if message == "bye" {
				return
			}

			if !assert.NoError(t, websocket.Message.Send(conn, "echo: "+message)) {
				return
			}
		}
	})

	echo.Backend = integration.NewBackend(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/refuse" {
			writer.Header().Set("Content-Type", "text/plain")
			writer.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(writer, "denied")

			return
		}

		if request.Header.Get("Upgrade") == "" {
			writer.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(writer, "plain")

			return
		}

		handler.ServeHTTP(writer, request)
	})

	return echo
}

func (e *echoServer) opened(request *http.Request) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.origins = append(e.origins, request.Header.Get("Origin"))
}

func (e *echoServer) lastOrigin() string {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.origins) == 0 {
		return ""
	}

	return e.origins[len(e.origins)-1]
}

func (e *echoServer) markClosed(path string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.closed[path]++
}

func (e *echoServer) closedTimes(path string) int {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.closed[path]
}

func (e *echoServer) waitClosed(t *testing.T, path string, times int) {
	t.Helper()

	require.Eventually(t, func() bool {
		return e.closedTimes(path) == times
	}, wait, pollTick, "upstream did not see %s closed", path)
}

func dial(t *testing.T, env *integration.Env, host, path string) *websocket.Conn {
	t.Helper()

	target, err := url.Parse(env.URL(host, path))
	require.NoError(t, err)

	location := *target
	location.Scheme = map[string]string{"http": "ws", "https": "wss"}[target.Scheme]

	wsConfig, err := websocket.NewConfig(location.String(), target.Scheme+"://"+target.Host)
	require.NoError(t, err)

	conn, err := env.Hosts.DialContext(t.Context(), "tcp", target.Host)
	require.NoError(t, err)

	var transport io.ReadWriteCloser = conn

	if target.Scheme == "https" {
		httpTransport, ok := env.Client.Transport.(*http.Transport)
		require.True(t, ok)

		tlsConfig := httpTransport.TLSClientConfig.Clone()
		tlsConfig.ServerName = host
		transport = tls.Client(conn, tlsConfig)
	}

	socket, err := websocket.NewClient(wsConfig, transport)
	require.NoError(t, err)
	require.NoError(t, socket.SetDeadline(time.Now().Add(wait)))

	t.Cleanup(func() { _ = socket.Close() })

	return socket
}

func exchange(t *testing.T, socket *websocket.Conn, message string) {
	t.Helper()

	require.NoError(t, websocket.Message.Send(socket, message))

	var reply string

	require.NoError(t, websocket.Message.Receive(socket, &reply))
	assert.Equal(t, "echo: "+message, reply)
}

func tunnels(env *integration.Env, rawURL string) []events.RequestCompleted {
	var found []events.RequestCompleted

	for _, event := range env.Events() {
		typed, ok := event.(events.RequestCompleted)
		if ok && urlt.URL_String(typed.Data.URL) == rawURL && typed.Data.Code == http.StatusSwitchingProtocols {
			found = append(found, typed)
		}
	}

	return found
}

func assertTunnelCompleted(t *testing.T, env *integration.Env, host, path string, times int) {
	t.Helper()

	rawURL := env.URL(host, path)

	require.Eventually(t, func() bool {
		return len(tunnels(env, rawURL)) == times
	}, wait, pollTick, "no RequestCompleted with 101 for %s", rawURL)

	for _, event := range tunnels(env, rawURL) {
		assert.Equal(t, events.HandlerProxy, event.Handler)
		assert.NoError(t, event.Err)
	}
}

func TestWebSocket(t *testing.T) {
	echo := newEchoServer(t)
	env := integration.New(t, echo.Backend, &config.UncorsConfig{
		Mappings: config.Mappings{
			{From: hosts.Parse("http://ws.local"), To: echo.AsHost()},
			{From: hosts.Parse("https://wss.local"), To: echo.AsHost()},
		},
	})

	for _, host := range []string{"ws.local", "wss.local"} {
		t.Run("exchanges messages both ways via "+host, func(t *testing.T) {
			socket := dial(t, env, host, "/exchange")

			exchange(t, socket, "hello")
			exchange(t, socket, "world")

			assert.Equal(t, echo.URL(), echo.lastOrigin())
		})

		t.Run("closes the tunnel when the client closes via "+host, func(t *testing.T) {
			path := "/client-close/" + host
			socket := dial(t, env, host, path)

			exchange(t, socket, "ping")
			require.NoError(t, socket.Close())

			echo.waitClosed(t, path, 1)
			assertTunnelCompleted(t, env, host, path, 1)
		})

		t.Run("closes the tunnel when the upstream closes via "+host, func(t *testing.T) {
			path := "/upstream-close/" + host
			socket := dial(t, env, host, path)

			exchange(t, socket, "ping")
			require.NoError(t, websocket.Message.Send(socket, "bye"))

			var reply string

			require.ErrorIs(t, websocket.Message.Receive(socket, &reply), io.EOF)
			assertTunnelCompleted(t, env, host, path, 1)
		})

		t.Run("returns a refusal as a regular response via "+host, func(t *testing.T) {
			request := integration.NewRequest(t, http.MethodGet, env.URL(host, "/refuse"))
			request.Header.Set("Connection", "Upgrade")
			request.Header.Set("Upgrade", "websocket")
			request.Header.Set("Sec-WebSocket-Version", "13")
			request.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
			request.Header.Set("Origin", "https://app.local")

			result, tracked := env.DoTracked(t, request)

			assert.Equal(t, http.StatusForbidden, result.Response.StatusCode)
			assert.Equal(t, "denied", result.BodyString())
			assert.Equal(t, "https://app.local", result.Response.Header.Get("Access-Control-Allow-Origin"))
			assert.Equal(t, http.StatusForbidden, tracked.Completed.Data.Code)
		})
	}
}

func TestWebSocketSkipsHARAndCache(t *testing.T) {
	harPath := filepath.Join(t.TempDir(), "out.har")

	echo := newEchoServer(t)
	env := integration.New(t, echo.Backend, &config.UncorsConfig{
		CacheConfig: config.CacheConfig{
			Methods:        []string{http.MethodGet},
			MaxSize:        10 * 1024 * 1024,
			ExpirationTime: 5 * time.Minute,
		},
		Mappings: config.Mappings{{
			From:  hosts.Parse("https://recorded.local"),
			To:    echo.AsHost(),
			HAR:   config.HARConfig{File: harPath},
			Cache: config.CacheGlobs{"/**"},
		}},
	})

	plain := func() {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("recorded.local", "/socket")))
		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "plain", result.BodyString())
	}

	plain()

	for index := range 2 {
		socket := dial(t, env, "recorded.local", "/socket")
		exchange(t, socket, "hello")
		require.NoError(t, socket.Close())
		echo.waitClosed(t, "/socket", index+1)
		assertTunnelCompleted(t, env, "recorded.local", "/socket", index+1)

		assert.Equal(t, index+2, echo.Count())
	}

	plain()

	assert.Equal(t, 3, echo.Count())

	type harEntry struct {
		URL    string
		Status int
	}

	var entries []harEntry

	require.Eventually(t, func() bool {
		data, err := afero.ReadFile(env.FS, harPath)
		if err != nil {
			return false
		}

		var parsed struct {
			Log struct {
				Entries []struct {
					Request struct {
						URL string `json:"url"`
					} `json:"request"`
					Response struct {
						Status int `json:"status"`
					} `json:"response"`
				} `json:"entries"`
			} `json:"log"`
		}

		if json.Unmarshal(data, &parsed) != nil {
			return false
		}

		entries = entries[:0]
		for _, entry := range parsed.Log.Entries {
			entries = append(entries, harEntry{URL: entry.Request.URL, Status: entry.Response.Status})
		}

		return len(entries) >= 2
	}, wait, pollTick)

	expected := harEntry{URL: env.URL("recorded.local", "/socket"), Status: http.StatusOK}
	assert.Equal(t, []harEntry{expected, expected}, entries)
}

func TestWebSocketServerShutdown(t *testing.T) {
	echo := newEchoServer(t)
	env := integration.New(t, echo.Backend, &config.UncorsConfig{
		Mappings: config.Mappings{
			{From: hosts.Parse("http://ws.local"), To: echo.AsHost()},
			{From: hosts.Parse("https://wss.local"), To: echo.AsHost()},
		},
	})

	plain := dial(t, env, "ws.local", "/plain")
	secure := dial(t, env, "wss.local", "/secure")

	exchange(t, plain, "hello")
	exchange(t, secure, "hello")

	env.Shutdown(t)

	requireClosedByProxy(t, plain)
	requireClosedByProxy(t, secure)

	echo.waitClosed(t, "/plain", 1)
	echo.waitClosed(t, "/secure", 1)
	assertTunnelCompleted(t, env, "ws.local", "/plain", 1)
	assertTunnelCompleted(t, env, "wss.local", "/secure", 1)
}

func TestWebSocketReload(t *testing.T) {
	echo := newEchoServer(t)
	env := integration.New(t, echo.Backend, &config.UncorsConfig{
		Mappings: config.Mappings{
			{From: hosts.Parse("http://ws.local"), To: echo.AsHost()},
			{From: hosts.Parse("https://wss.local"), To: echo.AsHost()},
		},
	})

	plain := dial(t, env, "ws.local", "/plain")
	secure := dial(t, env, "wss.local", "/secure")

	exchange(t, plain, "hello")
	exchange(t, secure, "hello")

	env.Reload(t)

	requireClosedByProxy(t, plain)
	requireClosedByProxy(t, secure)

	echo.waitClosed(t, "/plain", 1)
	echo.waitClosed(t, "/secure", 1)
	assertTunnelCompleted(t, env, "ws.local", "/plain", 1)
	assertTunnelCompleted(t, env, "wss.local", "/secure", 1)

	exchange(t, dial(t, env, "ws.local", "/after"), "again")
	exchange(t, dial(t, env, "wss.local", "/after"), "again")
}

func requireClosedByProxy(t *testing.T, socket *websocket.Conn) {
	t.Helper()

	var reply string

	err := websocket.Message.Receive(socket, &reply)
	require.Error(t, err)

	var netErr net.Error
	if errors.As(err, &netErr) {
		require.False(t, netErr.Timeout(), "the proxy did not close the connection: %v", err)
	}
}

func TestWebSocketOutsideAllowedOrigins(t *testing.T) {
	echo := newEchoServer(t)
	env := integration.New(t, echo.Backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From:           hosts.Parse("https://wss.local"),
			To:             echo.AsHost(),
			AllowedOrigins: config.AllowedOrigins{"http://localhost:*"},
		}},
	})

	t.Run("still connects", func(t *testing.T) {
		socket := dial(t, env, "wss.local", "/rejected")

		exchange(t, socket, "hello")

		assert.Equal(t, echo.URL(), echo.lastOrigin())
	})

	t.Run("answers the handshake without CORS headers", func(t *testing.T) {
		request := integration.NewRequest(t, http.MethodGet, env.URL("wss.local", "/handshake"))
		request.Header.Set("Connection", "Upgrade")
		request.Header.Set("Upgrade", "websocket")
		request.Header.Set("Sec-WebSocket-Version", "13")
		request.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
		request.Header.Set("Origin", "https://app.local")

		response, err := env.Client.Do(request)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())

		assert.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)
		assert.Equal(t, "websocket", response.Header.Get("Upgrade"))
		assert.Empty(t, response.Header.Get("Access-Control-Allow-Origin"))
		assert.Contains(t, response.Header.Values("Vary"), "Origin")

		echo.waitClosed(t, "/handshake", 1)
		assertTunnelCompleted(t, env, "wss.local", "/handshake", 1)
	})
}
