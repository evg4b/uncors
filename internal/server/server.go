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
	"github.com/hashicorp/go-multierror"
	"github.com/samber/lo"
)

const (
	readHeaderTimeout = 30 * time.Second
	shutdownTimeout   = 15 * time.Second
)

type Target struct {
	Address   string
	TLSConfig *tls.Config
	Handler   contracts.Handler
}

type Server struct {
	sync.WaitGroup

	listeners []*PortListener
}

func New() *Server {
	return &Server{
		listeners: []*PortListener{},
	}
}

func (s *Server) Start(ctx context.Context, targets []Target) error {
	s.listeners = lo.Map(targets, func(target Target, _ int) *PortListener {
		portCtx, portCtxCancel := context.WithCancel(ctx)

		portListener := &PortListener{
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

		portListener.RegisterOnShutdown(portCtxCancel)

		return portListener
	})

	var launchWaitGroup sync.WaitGroup
	launchWaitGroup.Add(len(s.listeners))

	var (
		errMu        sync.Mutex
		launchErrors *multierror.Error
	)

	for _, server := range s.listeners {
		s.Go(func() {
			var startupFailed bool

			err := server.Listen(ctx, func(listenErr error) {
				if listenErr != nil {
					startupFailed = true

					errMu.Lock()

					launchErrors = multierror.Append(launchErrors, listenErr)

					errMu.Unlock()
				}

				launchWaitGroup.Done()
			})

			if !startupFailed && err != nil && !errors.Is(err, http.ErrServerClosed) {
				errMu.Lock()

				launchErrors = multierror.Append(launchErrors, err)

				errMu.Unlock()
			}
		})
	}

	launchWaitGroup.Wait()

	return launchErrors.ErrorOrNil()
}

func (s *Server) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	var waitGroup sync.WaitGroup

	var errors *multierror.Error

	for _, server := range s.listeners {
		waitGroup.Go(func() {
			err := server.Shutdown(ctx)
			if err != nil {
				errors = multierror.Append(errors, err)
			}
		})
	}

	waitGroup.Wait()

	return errors.ErrorOrNil()
}

func (s *Server) Restart(ctx context.Context, targets []Target) error {
	err := s.Shutdown(ctx)
	if err != nil {
		return err
	}

	return s.Start(ctx, targets)
}

func (s *Server) Wait() {
	s.WaitGroup.Wait()
}

func (s *Server) Close() error {
	var errors *multierror.Error

	for _, pl := range s.listeners {
		err := pl.Close()
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}

	return errors.ErrorOrNil()
}
