package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/evg4b/uncors/internal/contracts"
)

// PortListener serves one address. Its handler is swappable so that a config
// reload can replace the routing graph of an already listening port without
// closing the socket and dropping live connections.
type PortListener struct {
	http.Server

	address   string
	enableTLS bool
	manager   *HostCertManager
	handler   atomic.Pointer[contracts.Handler]
}

func (ps *PortListener) SetHandler(handler contracts.Handler) {
	ps.handler.Store(&handler)
}

func (ps *PortListener) Handler() contracts.Handler {
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
			MinVersion:     tls.VersionTLS12,
			GetCertificate: ps.manager.getCertificate,
		})
	}

	onReady()

	return ps.Serve(listener)
}
