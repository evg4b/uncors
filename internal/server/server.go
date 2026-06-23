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
	"github.com/evg4b/uncors/internal/infra"
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
	sink      RequestSink
	nextID    atomic.Uint64
}

// New creates a server that reports request activity to sink. A nil sink is
// replaced by NoopRequestSink so the request path is never left without one.
func New(manager *HostCertManager, sink RequestSink) *Server {
	if sink == nil {
		sink = NoopRequestSink{}
	}

	return &Server{
		listeners: []*PortListener{},
		manager:   manager,
		sink:      sink,
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
		launchErrorsMu sync.Mutex
		launchErrs     []error
	)

	for _, server := range s.listeners {
		s.Add(1)
		go func(srv *PortListener) {
			defer s.Done()

			var once sync.Once

			err := srv.Listen(ctx, func() {
				once.Do(launchWaitGroup.Done)
			})

			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				launchErrorsMu.Lock()

				launchErrs = append(launchErrs, err)
				launchErrorsMu.Unlock()
			}
			// On bind failure onReady is never called, so we must release the
			// wait group here (after recording the error). On a clean shutdown
			// onReady already fired, so this is a no-op.
			once.Do(launchWaitGroup.Done)
		}(server)
	}

	launchWaitGroup.Wait()

	launchErrorsMu.Lock()
	err := errors.Join(launchErrs...)
	launchErrorsMu.Unlock()

	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	var (
		waitGroup sync.WaitGroup
		errsMu    sync.Mutex
		errs      []error
	)

	for _, server := range s.listeners {
		waitGroup.Add(1)
		go func(srv *PortListener) {
			defer waitGroup.Done()

			err := srv.Shutdown(ctx)
			if err != nil {
				errsMu.Lock()
				defer errsMu.Unlock()

				errs = append(errs, err)
			}
		}(server)
	}

	waitGroup.Wait()

	return errors.Join(errs...)
}

func (s *Server) Restart(ctx context.Context, targets []Target) error {
	s.Add(1)
	defer s.Done()

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
	var errs []error

	for _, portListener := range s.listeners {
		err := portListener.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (s *Server) handleRequest(handler contracts.Handler, writer http.ResponseWriter, request *http.Request) {
	helpers.NormaliseRequest(request)

	rec := NewResponseRecorder(writer)
	requestID := s.nextID.Add(1)

	s.sink.Emit(RequestEvent{
		ID:        requestID,
		Method:    request.Method,
		URL:       request.URL,
		StartedAt: time.Now(),
	})

	// The prefix is reported once, with the terminal event. Handlers may set it
	// from a goroutine of their own, so access is guarded.
	var (
		prefixMu   sync.Mutex
		lastPrefix string
	)

	ctx := context.WithValue(request.Context(), contracts.PrefixUpdaterKey, func(prefix string) {
		prefixMu.Lock()
		defer prefixMu.Unlock()

		lastPrefix = prefix
	})

	err := handler.ServeHTTP(rec, request.WithContext(ctx))
	if err != nil {
		infra.HTTPError(rec, err)
	}

	data := helpers.ToRequestData(request, helpers.NormaliseStatusCode(rec.StatusCode()))
	data.Cancelled = ctx.Err() != nil

	prefixMu.Lock()
	prefix := lastPrefix
	prefixMu.Unlock()

	s.sink.Emit(RequestEvent{
		ID:     requestID,
		Done:   true,
		Prefix: prefix,
		Data:   data,
	})
}
