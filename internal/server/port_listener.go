package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
)

type PortListener struct {
	http.Server

	target *Target
}

func (ps *PortListener) Listen(ctx context.Context, onListening func()) error {
	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(ctx, "tcp", ps.target.Address)
	if err != nil {
		onListening()

		return err
	}

	if ps.target.TLSConfig != nil {
		listener = tls.NewListener(listener, ps.target.TLSConfig)
	}

	onListening()

	return ps.Serve(listener)
}
