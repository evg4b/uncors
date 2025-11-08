package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
)

type PortListner struct {
	http.Server

	target *Target
}

func (ps *PortListner) Lister(ctx context.Context) error {
	var listenConfig net.ListenConfig

	listner, err := listenConfig.Listen(ctx, "tcp", ps.target.Address)
	if err != nil {
		return err
	}

	if ps.target.TLSConfgi != nil {
		listner = tls.NewListener(listner, ps.target.TLSConfgi)
	}

	err = ps.Serve(listner)
	if err != nil {
		return err
	}

	return nil
}

func (ps *PortListner) Shutdown(ctx context.Context) error {
	return ps.Server.Shutdown(ctx)
}
