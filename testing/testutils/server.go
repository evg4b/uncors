package testutils

import (
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type TestServer struct {
	URL      string
	listener net.Listener
	server   *http.Server
}

func (s *TestServer) Close() {
	_ = s.server.Close()
	_ = s.listener.Close()
}

// NewServer binds an IPv4 loopback listener to avoid environments where ::1 is unavailable.
func NewServer(t *testing.T, handler http.Handler) *TestServer {
	t.Helper()

	listener, err := net.Listen("tcp4", "127.0.0.1:0") //nolint:noctx
	require.NoError(t, err)

	server := &http.Server{Handler: handler}
	go func() {
		_ = server.Serve(listener)
	}()

	return &TestServer{
		URL:      "http://" + listener.Addr().String(),
		listener: listener,
		server:   server,
	}
}
