package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
)

type PortListener struct {
	http.Server

	target  *Target
	manager *HostCertManager
}

func (ps *PortListener) Listen(ctx context.Context, onReady func(error)) error {
	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(ctx, "tcp", ps.target.Address)
	if err != nil {
		onReady(err)

		return err
	}

	if ps.target.EnableTLS {
		listener = tls.NewListener(listener, &tls.Config{
			MinVersion:     tls.VersionTLS12,
			GetCertificate: ps.manager.getCertificate,
		})
	}

	onReady(nil)

	return ps.Serve(listener)
}
