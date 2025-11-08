package server

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/samber/lo"
)

const (
	readHeaderTimeout = 30 * time.Second
	shutdownTimeout   = 15 * time.Second
)

type Target struct {
	Address   string
	TLSConfgi *tls.Config
	Handler   contracts.Handler
}

type Server struct {
	waitGroup sync.WaitGroup
	servers   []*PortListner
}

func New() *Server {
	return &Server{
		waitGroup: sync.WaitGroup{},
		servers:   []*PortListner{},
	}
}

func (s *Server) Start(ctx context.Context, targets []Target) {
	s.servers = lo.Map(targets, func(target Target, _ int) *PortListner {
		portCtx, portCtxCancel := context.WithCancel(ctx)

		portListner := &PortListner{
			Server: http.Server{
				BaseContext: func(_ net.Listener) context.Context {
					return portCtx
				},
				ReadHeaderTimeout: readHeaderTimeout,
				Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					helpers.NormaliseRequest(request)
					target.Handler.ServeHTTP(contracts.WrapResponseWriter(writer), request)
				}),
			},
			target: &target,
		}

		portListner.RegisterOnShutdown(portCtxCancel)

		return portListner
	})

	var launchWaitGroup sync.WaitGroup
	launchWaitGroup.Add(len(s.servers))

	for _, server := range s.servers {
		s.waitGroup.Go(func() {
			launchWaitGroup.Done()

			err := server.Lister(ctx)
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		})
	}

	launchWaitGroup.Wait()
}

func (s *Server) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	var waitGroup sync.WaitGroup

	errChan := make(chan error, len(s.servers))

	for _, server := range s.servers {
		waitGroup.Go(func() {
			err := server.Shutdown(ctx)
			if err != nil {
				errChan <- err
			}
		})
	}

	waitGroup.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) Restart(ctx context.Context, targets []Target) error {
	err := s.Shutdown(ctx)
	if err != nil {
		return err
	}

	s.Start(ctx, targets)

	return nil
}

func (s *Server) Wait() {
	s.waitGroup.Wait()
}

func (s *Server) Close() error {
	var err error
	for _, pl := range s.servers {
		err = pl.Close()
	}

	return err
}
