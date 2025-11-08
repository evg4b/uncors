package server

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/samber/lo"
)

const (
	baseAddress       = "127.0.0.1"
	readHeaderTimeout = 30 * time.Second
	shutdownTimeout   = 15 * time.Second
)

type Server struct {
	servers []*PortListner
}

func New() *Server {
	return &Server{
		servers: []*PortListner{},
	}
}

func (s *Server) Start(ctx context.Context, groups []config.PortGroup) error {
	s.servers = lo.Map(groups, func(portGroup config.PortGroup, _ int) *PortListner {
		portCtx, portCtxCancel := context.WithCancel(ctx)

		portListner := PortListner{
			Server: http.Server{
				BaseContext: func(_ net.Listener) context.Context {
					return portCtx
				},
				ReadHeaderTimeout: readHeaderTimeout,
				Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					writer.WriteHeader(http.StatusOK)
					writer.Write([]byte("Tetst"))
				}),
			},
		}

		portListner.Server.RegisterOnShutdown(portCtxCancel)

		return &portListner
	})

	lo.ForEach(s.servers, func(server *PortListner, _ int) {
		go server.Lister(ctx)
	})

	return nil
}

func (s *Server) Waite() {
	<-make(chan int)
}
