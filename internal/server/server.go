package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
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
	Handler   contracts.Handler
	EnableTLS bool
}

type Server struct {
	sync.WaitGroup

	listeners []*PortListener
	manager   *HostCertManager
	tracker   *RequestTracker
	nextID    atomic.Uint64
}

func New(manager *HostCertManager, tracker *RequestTracker) *Server {
	return &Server{
		listeners: []*PortListener{},
		manager:   manager,
		tracker:   tracker,
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
					s.handleRequest(target.Handler, writer, request)
				}),
			},
			target:  &target,
			manager: s.manager,
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
					defer errMu.Unlock()

					launchErrors = multierror.Append(launchErrors, listenErr)
				}

				launchWaitGroup.Done()
			})

			if !startupFailed && err != nil && !errors.Is(err, http.ErrServerClosed) {
				errMu.Lock()
				defer errMu.Unlock()

				launchErrors = multierror.Append(launchErrors, err)
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

func (s *Server) handleRequest(handler contracts.Handler, writer http.ResponseWriter, request *http.Request) {
	helpers.NormaliseRequest(request)

	responseWriter := contracts.WrapResponseWriter(writer)

	requestID := s.nextID.Add(1)
	select {
	case s.tracker.events <- RequestEvent{
		ID:        requestID,
		Method:    request.Method,
		URL:       request.URL,
		StartedAt: time.Now(),
	}:
	default:
	}

	var lastPrefix string

	ctx := context.WithValue(request.Context(), contracts.PrefixUpdaterKey, func(p string) {
		lastPrefix = p
		select {
		case s.tracker.events <- RequestEvent{ID: requestID, Prefix: p}:
		default:
		}
	})

	handler.ServeHTTP(responseWriter, request.WithContext(ctx))

	data := helpers.ToRequestData(request, helpers.NormaliseStatusCode(responseWriter.StatusCode()))
	data.Cancelled = ctx.Err() != nil

	select {
	case s.tracker.events <- RequestEvent{ID: requestID, Done: true, Prefix: lastPrefix, Data: data}:
	default:
	}
}
