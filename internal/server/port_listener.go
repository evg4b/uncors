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

func (ps *PortListener) Listen(ctx context.Context) error {
	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(ctx, "tcp", ps.target.Address)
	if err != nil {
		return err
	}

	if ps.target.TLSConfig != nil {
		listener = tls.NewListener(listener, ps.target.TLSConfig)
	}

	err = ps.Serve(listener)
	if err != nil {
		return err
	}

	return nil
}

func (ps *PortListener) Shutdown(ctx context.Context) error {
	return ps.Server.Shutdown(ctx)
}
