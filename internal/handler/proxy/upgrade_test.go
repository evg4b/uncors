package proxy_test

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tunnelWait = 5 * time.Second

type hijackableRecorder struct {
	*httptest.ResponseRecorder

	conn net.Conn
}

func (r *hijackableRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return r.conn, bufio.NewReadWriter(bufio.NewReader(r.conn), bufio.NewWriter(r.conn)), nil
}

type tunnelCase struct {
	handler   *proxy.Handler
	request   *http.Request
	recorder  *hijackableRecorder
	writer    *server.ResponseRecorder
	client    net.Conn
	upstream  net.Conn
	forwarded *http.Request
	done      chan error
}

func upgradeRequest(t *testing.T) *http.Request {
	t.Helper()

	request := premiumRequest(t, http.MethodGet, nil)
	request.Header.Set("Connection", "Upgrade")
	request.Header.Set("Upgrade", "echo")
	request.Header.Set("Origin", "http://premium.local.com")
	request.Header.Set("Referer", "http://premium.local.com/app")

	return request
}

func upgradeHandler(response func(*http.Request) *http.Response, forwarded **http.Request) *proxy.Handler {
	return proxy.NewProxyHandler(
		proxy.WithHTTPClient(testutils.NewTestClient(func(req *http.Request) *http.Response {
			*forwarded = req
			upstreamResponse := response(req)
			upstreamResponse.Request = req

			return upstreamResponse
		})),
		proxy.WithURLReplacerFactory(lo.Must(urlreplacer.NewURLReplacerFactory(config.Mappings{
			{From: hosts.Parse("http://premium.local.com"), To: hosts.Parse("https://premium.api.com")},
		}))),
	)
}

func startTunnel(t *testing.T, request *http.Request) *tunnelCase {
	t.Helper()

	client, proxyClient := net.Pipe()
	proxyUpstream, upstream := net.Pipe()

	t.Cleanup(func() {
		_ = client.Close()
		_ = upstream.Close()
	})

	for _, conn := range []net.Conn{client, upstream} {
		require.NoError(t, conn.SetDeadline(time.Now().Add(tunnelWait)))
	}

	testCase := &tunnelCase{
		request:  request,
		recorder: &hijackableRecorder{ResponseRecorder: httptest.NewRecorder(), conn: proxyClient},
		client:   client,
		upstream: upstream,
		done:     make(chan error, 1),
	}

	testCase.handler = upgradeHandler(func(*http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusSwitchingProtocols,
			Header: http.Header{
				"Connection":                  {"Upgrade"},
				"Upgrade":                     {"echo"},
				"Sec-Websocket-Accept":        {"accepted"},
				"Access-Control-Allow-Origin": {"*"},
			},
			Body: proxyUpstream,
		}
	}, &testCase.forwarded)

	testCase.writer = server.NewResponseRecorder(testCase.recorder)

	go func() {
		testCase.done <- testCase.handler.ServeHTTP(testCase.writer, request)
	}()

	return testCase
}

func (c *tunnelCase) wait(t *testing.T) {
	t.Helper()

	select {
	case err := <-c.done:
		require.NoError(t, err)
	case <-time.After(tunnelWait):
		t.Fatalf("tunnel did not close within %s", tunnelWait)
	}
}

func transfer(t *testing.T, from, to net.Conn, message string) {
	t.Helper()

	go func() {
		_, err := io.WriteString(from, message)
		assert.NoError(t, err)
	}()

	received := make([]byte, len(message))
	_, err := io.ReadFull(to, received)
	require.NoError(t, err)
	assert.Equal(t, message, string(received))
}

func assertClosed(t *testing.T, conn net.Conn) {
	t.Helper()

	_, err := conn.Read(make([]byte, 1))
	assert.ErrorIs(t, err, io.EOF)
}

func TestProxyHandlerUpgrade(t *testing.T) {
	t.Run("forwards the handshake and copies data both ways", func(t *testing.T) {
		testCase := startTunnel(t, upgradeRequest(t))

		transfer(t, testCase.client, testCase.upstream, "ping")
		transfer(t, testCase.upstream, testCase.client, "pong")
		transfer(t, testCase.client, testCase.upstream, "again")

		require.NoError(t, testCase.client.Close())
		testCase.wait(t)

		assert.Equal(t, "https://premium.api.com/", testCase.forwarded.URL.String())
		assert.Equal(t, "Upgrade", testCase.forwarded.Header.Get("Connection"))
		assert.Equal(t, "echo", testCase.forwarded.Header.Get("Upgrade"))
		assert.Equal(t, "https://premium.api.com", testCase.forwarded.Header.Get("Origin"))
		assert.Equal(t, "https://premium.api.com/app", testCase.forwarded.Header.Get("Referer"))

		assert.Equal(t, http.StatusSwitchingProtocols, testCase.writer.StatusCode())
		assert.Equal(t, http.StatusSwitchingProtocols, testCase.recorder.Code)

		header := testCase.recorder.Header()
		assert.Equal(t, "echo", header.Get("Upgrade"))
		assert.Equal(t, "Upgrade", header.Get("Connection"))
		assert.Equal(t, "accepted", header.Get("Sec-Websocket-Accept"))
		assert.Equal(t, "http://premium.local.com", header.Get("Access-Control-Allow-Origin"))
	})

	t.Run("withholds CORS headers from a rejected origin", func(t *testing.T) {
		testCase := startTunnel(t, testutils.RejectOrigin(t, upgradeRequest(t)))

		transfer(t, testCase.client, testCase.upstream, "ping")
		require.NoError(t, testCase.client.Close())
		testCase.wait(t)

		header := testCase.recorder.Header()
		assert.Equal(t, http.StatusSwitchingProtocols, testCase.recorder.Code)
		assert.Equal(t, "echo", header.Get("Upgrade"))
		assert.Empty(t, header.Get("Access-Control-Allow-Origin"))
		assert.Empty(t, header.Get("Access-Control-Allow-Credentials"))
		assert.Contains(t, header.Values("Vary"), "Origin")
	})

	t.Run("closes the upstream when the client closes", func(t *testing.T) {
		testCase := startTunnel(t, upgradeRequest(t))

		require.NoError(t, testCase.client.Close())
		testCase.wait(t)

		assertClosed(t, testCase.upstream)
	})

	t.Run("closes the client when the upstream closes", func(t *testing.T) {
		testCase := startTunnel(t, upgradeRequest(t))

		transfer(t, testCase.upstream, testCase.client, "bye")
		require.NoError(t, testCase.upstream.Close())
		testCase.wait(t)

		assertClosed(t, testCase.client)
	})

	t.Run("closes both sides when the request is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		testCase := startTunnel(t, upgradeRequest(t).WithContext(ctx))

		transfer(t, testCase.client, testCase.upstream, "ping")
		cancel()
		testCase.wait(t)

		assertClosed(t, testCase.client)
		assertClosed(t, testCase.upstream)
	})

	t.Run("returns a refusal as a regular response", func(t *testing.T) {
		var forwarded *http.Request

		handler := upgradeHandler(func(*http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     http.Header{"Content-Type": {"text/plain"}},
				Body:       io.NopCloser(strings.NewReader("denied")),
			}
		}, &forwarded)

		recorder := httptest.NewRecorder()

		require.NoError(t, handler.ServeHTTP(server.NewResponseRecorder(recorder), upgradeRequest(t)))

		assert.Equal(t, http.StatusForbidden, recorder.Code)
		assert.Equal(t, "denied", recorder.Body.String())
		assert.Equal(t, "http://premium.local.com", recorder.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("fails when the upstream connection is not writable", func(t *testing.T) {
		var forwarded *http.Request

		handler := upgradeHandler(func(*http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusSwitchingProtocols,
				Header:     http.Header{"Connection": {"Upgrade"}, "Upgrade": {"echo"}},
				Body:       io.NopCloser(strings.NewReader("")),
			}
		}, &forwarded)

		writer := server.NewResponseRecorder(httptest.NewRecorder())

		require.Error(t, handler.ServeHTTP(writer, upgradeRequest(t)))
		assert.Zero(t, writer.StatusCode())
	})

	t.Run("closes the upstream when the client connection cannot be taken over", func(t *testing.T) {
		var forwarded *http.Request

		proxyUpstream, upstream := net.Pipe()
		require.NoError(t, upstream.SetDeadline(time.Now().Add(tunnelWait)))

		handler := upgradeHandler(func(*http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusSwitchingProtocols,
				Header:     http.Header{"Connection": {"Upgrade"}, "Upgrade": {"echo"}},
				Body:       proxyUpstream,
			}
		}, &forwarded)

		err := handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), upgradeRequest(t))

		require.ErrorIs(t, err, http.ErrNotSupported)
		assertClosed(t, upstream)
	})
}
