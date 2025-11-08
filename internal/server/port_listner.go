package server

import (
	"context"
	"net"
	"net/http"
	"strconv"
)

type PortListner struct {
	http.Server

	port int
}

func (ps *PortListner) Lister(ctx context.Context) error {
	var listenConfig net.ListenConfig

	address := net.JoinHostPort(baseAddress, strconv.Itoa(ps.port))

	listner, err := listenConfig.Listen(ctx, "tcp", address)
	if err != nil {
		return err
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
