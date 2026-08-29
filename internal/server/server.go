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
	"github.com/evg4b/uncors/internal/infra"
	"github.com/samber/lo"
)

const (
	readHeaderTimeout = 30 * time.Second
	shutdownTimeout   = 15 * time.Second
)

type Target struct {
	Address   string
	Handler   http.Handler
	EnableTLS bool
	// DefaultHost names the certificate to serve to a TLS client that sends no
	// SNI. A wildcard bind cannot infer it from the connection's local address.
	DefaultHost string
}

type Server struct {
	sync.WaitGroup

	mu        sync.Mutex
	listeners map[string]*PortListener

	manager *HostCertManager
	sink    RequestSink
	nextID  atomic.Uint64
}

// New creates a server that reports request activity to sink. A nil sink is
// replaced by NoopRequestSink so the request path is never left without one.
func New(manager *HostCertManager, sink RequestSink) *Server {
	if sink == nil {
		sink = NoopRequestSink{}
	}

	return &Server{
		listeners: map[string]*PortListener{},
		manager:   manager,
		sink:      sink,
	}
}

// Start brings the given targets online. It is idempotent with respect to
// addresses that are already served: such a port keeps its listener and only
// swaps its handler, so applying a new configuration never drops connections on
// a port whose address did not change.
func (s *Server) Start(ctx context.Context, targets []Target) error {
	obsolete, pending := s.plan(targets)

	return errors.Join(
		shutdownListeners(ctx, obsolete),
		s.launch(ctx, pending),
	)
}

// Restart applies a new set of targets to a running server. Ports common to the
// old and the new configuration stay up throughout.
func (s *Server) Restart(ctx context.Context, targets []Target) error {
	return s.Start(ctx, targets)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return shutdownListeners(ctx, s.takeListeners())
}

func (s *Server) Wait() {
	s.WaitGroup.Wait()
}

func (s *Server) Close() error {
	var errs []error

	for _, portListener := range s.takeListeners() {
		err := portListener.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// plan reconciles the desired targets with the live listeners: listeners that
// remain desired get the new handler, listeners that are no longer desired are
// returned for shutdown, and listeners for new addresses are created (but not
// yet started) and returned as pending.
func (s *Server) plan(targets []Target) ([]*PortListener, []*PortListener) {
	s.mu.Lock()
	defer s.mu.Unlock()

	desired := make(map[string]Target, len(targets))
	for _, target := range targets {
		desired[target.Address] = target
	}

	var obsolete []*PortListener

	pending := make([]*PortListener, 0, len(desired))

	for address, listener := range s.listeners {
		target, keep := desired[address]
		if keep && target.EnableTLS == listener.enableTLS {
			listener.SetHandler(target.Handler)
			listener.defaultHost = target.DefaultHost

			delete(desired, address)

			continue
		}

		obsolete = append(obsolete, listener)

		delete(s.listeners, address)
	}

	for address, target := range desired {
		listener := s.newPortListener(target)
		s.listeners[address] = listener
		pending = append(pending, listener)
	}

	return obsolete, pending
}

func (s *Server) newPortListener(target Target) *PortListener {
	portListener := &PortListener{
		address:     target.Address,
		enableTLS:   target.EnableTLS,
		defaultHost: target.DefaultHost,
		manager:     s.manager,
	}

	portListener.SetHandler(target.Handler)

	portListener.Server = http.Server{
		ReadHeaderTimeout: readHeaderTimeout,
		// Without this, net/http writes TLS handshake failures and recovered
		// handler panics to the standard logger, where they used to be
		// discarded.
		ErrorLog: infra.StdLogger(),
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			s.handleRequest(portListener.Handler(), writer, request)
		}),
	}

	return portListener
}

// launch starts the pending listeners and blocks until each of them is either
// serving or has failed to bind.
func (s *Server) launch(ctx context.Context, pending []*PortListener) error {
	if len(pending) == 0 {
		return nil
	}

	err := s.checkTLSReadiness(pending)
	if err != nil {
		return err
	}

	var launchWaitGroup sync.WaitGroup

	launchWaitGroup.Add(len(pending))

	var (
		launchErrorsMu sync.Mutex
		launchErrs     []error
	)

	for _, listener := range pending {
		portCtx, portCtxCancel := context.WithCancel(ctx)

		listener.BaseContext = func(net.Listener) context.Context { return portCtx }
		listener.RegisterOnShutdown(portCtxCancel)

		s.Add(1)

		go func(srv *PortListener) {
			defer s.Done()

			var once sync.Once

			err := srv.Listen(ctx, func() {
				once.Do(launchWaitGroup.Done)
			})

			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				// A listener that never bound must not stay registered, or the
				// next reload would believe its port is already served.
				s.forget(srv)

				launchErrorsMu.Lock()

				launchErrs = append(launchErrs, err)
				launchErrorsMu.Unlock()
			}
			// On bind failure onReady is never called, so we must release the
			// wait group here (after recording the error). On a clean shutdown
			// onReady already fired, so this is a no-op.
			once.Do(launchWaitGroup.Done)
		}(listener)
	}

	launchWaitGroup.Wait()

	launchErrorsMu.Lock()
	defer launchErrorsMu.Unlock()

	return errors.Join(launchErrs...)
}

// takeListeners detaches every live listener from the server and returns them.
// checkTLSReadiness reports the missing local CA before any socket is opened, so
// that an https mapping fails with an actionable message rather than with a
// handshake error on the first request.
func (s *Server) checkTLSReadiness(pending []*PortListener) error {
	var errs []error

	for _, listener := range pending {
		if !listener.enableTLS {
			continue
		}

		if !CAExists(s.manager.fs) {
			errs = append(errs, &TLSError{Host: listener.address})

			s.forget(listener)
		}
	}

	return errors.Join(errs...)
}

func (s *Server) takeListeners() []*PortListener {
	s.mu.Lock()
	defer s.mu.Unlock()

	listeners := lo.Values(s.listeners)
	s.listeners = map[string]*PortListener{}

	return listeners
}

func (s *Server) forget(listener *PortListener) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listeners[listener.address] == listener {
		delete(s.listeners, listener.address)
	}
}

func shutdownListeners(ctx context.Context, listeners []*PortListener) error {
	if len(listeners) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	var (
		waitGroup sync.WaitGroup
		errsMu    sync.Mutex
		errs      []error
	)

	for _, listener := range listeners {
		waitGroup.Go(func() {
			err := listener.Shutdown(ctx)
			if err != nil {
				errsMu.Lock()
				defer errsMu.Unlock()

				errs = append(errs, err)
			}
		})
	}

	waitGroup.Wait()

	return errors.Join(errs...)
}

func (s *Server) handleRequest(handler http.Handler, writer http.ResponseWriter, request *http.Request) {
	infra.NormaliseRequest(request)

	rec := infra.NewResponseRecorder(writer)
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

	handler.ServeHTTP(rec, request.WithContext(ctx))

	data := infra.ToRequestData(request, infra.NormaliseStatusCode(rec.StatusCode()))
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
