package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync/atomic"
)

// PortListener serves one address. Its handler is swappable so that a config
// reload can replace the routing graph of an already listening port without
// closing the socket and dropping live connections.
type PortListener struct {
	http.Server

	address     string
	enableTLS   bool
	defaultHost string
	manager     *HostCertManager
	handler     atomic.Pointer[http.Handler]
}

func (ps *PortListener) SetHandler(handler http.Handler) {
	ps.handler.Store(&handler)
}

func (ps *PortListener) Handler() http.Handler {
	return *ps.handler.Load()
}

func (ps *PortListener) Listen(ctx context.Context, onReady func()) error {
	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(ctx, "tcp", ps.address)
	if err != nil {
		return err
	}

	if ps.enableTLS {
		listener = tls.NewListener(listener, &tls.Config{
			MinVersion: tls.VersionTLS12,
			GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return ps.manager.certificateFor(clientHello, ps.defaultHost)
			},
		})
	}

	onReady()

	return ps.Serve(listener)
}
