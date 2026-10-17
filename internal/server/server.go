package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/tracking"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

var errHandlerPanicked = errors.New("handler panicked")

const (
	readHeaderTimeout = 30 * time.Second
	shutdownTimeout   = 15 * time.Second
)

type Target struct {
	Address   string
	Handler   http.Handler
	EnableTLS bool
}

type listenerKey struct {
	address string
	tls     bool
}

func (t Target) key() listenerKey {
	return listenerKey{address: t.Address, tls: t.EnableTLS}
}

type route struct {
	handler  http.Handler
	requests *requests
}

type connKey struct{}

type Server struct {
	wg sync.WaitGroup

	mu              sync.Mutex
	listeners       map[listenerKey]*portListener
	requests        *requests
	retiring        map[*requests]struct{}
	closing         map[*portListener]struct{}
	getCertificate  func(*tls.ClientHelloInfo) (*tls.Certificate, error)
	publisher       events.Publisher
	nextID          atomic.Uint64
	shutdownTimeout time.Duration
}

func New(getCertificate func(*tls.ClientHelloInfo) (*tls.Certificate, error), publisher events.Publisher) *Server {
	return &Server{
		listeners:       map[listenerKey]*portListener{},
		retiring:        map[*requests]struct{}{},
		closing:         map[*portListener]struct{}{},
		getCertificate:  getCertificate,
		publisher:       publisher,
		shutdownTimeout: shutdownTimeout,
	}
}

func (s *Server) Start(ctx context.Context, targets []Target) error {
	return telemetry.Run(ctx, "server.start", func(spanCtx context.Context) error {
		_, err := s.apply(ctx, spanCtx, targets)

		return err
	})
}

func (s *Server) Shutdown(ctx context.Context) error {
	return telemetry.Run(ctx, "server.shutdown", s.shutdown)
}

func (s *Server) Restart(ctx context.Context, targets []Target) (<-chan struct{}, error) {
	s.wg.Add(1)
	defer s.wg.Done()

	var retired <-chan struct{}

	err := telemetry.Run(ctx, "server.restart", func(spanCtx context.Context) error {
		var err error

		retired, err = s.apply(ctx, spanCtx, targets)

		return err
	})

	return retired, err
}

func (s *Server) Wait() {
	s.wg.Wait()
}

func (s *Server) Close() error {
	listeners, retiring, current := s.release()

	var errs []error

	for _, portListener := range listeners {
		err := portListener.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	abort(retiring)

	if current != nil {
		current.retire()
	}

	return errors.Join(errs...)
}

// ctx is the listeners' base context. spanCtx carries the server.start span,
// which only the listen spans nest under: TLS handshakes and requests derive
// their contexts from the base one.
//
// Every target is bound before any is served, so a start that fails leaves
// nothing listening.
func (s *Server) apply(ctx, spanCtx context.Context, targets []Target) (<-chan struct{}, error) {
	trace.SpanFromContext(spanCtx).SetAttributes(attribute.Int("uncors.listeners", len(targets)))

	s.mu.Lock()
	defer s.mu.Unlock()

	held := make(map[string]*portListener, len(s.listeners))
	for _, portListener := range s.listeners {
		held[portListener.target.Address] = portListener
	}

	fresh := make(map[listenerKey]*portListener, len(targets))
	switched := make([]Target, 0, len(targets))

	var errs []error

	for _, target := range targets {
		if _, ok := s.listeners[target.key()]; ok {
			continue
		}

		if _, ok := held[target.Address]; ok {
			switched = append(switched, target)

			continue
		}

		err := s.bindFresh(ctx, spanCtx, target, fresh)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		closeListeners(fresh)

		return nil, errors.Join(errs...)
	}

	released := make([]*portListener, 0, len(switched))

	for _, target := range switched {
		previous := held[target.Address]
		released = append(released, previous)
		s.retireListener(ctx, previous)

		err := s.bindFresh(ctx, spanCtx, target, fresh)
		if err != nil {
			closeListeners(fresh)

			return nil, errors.Join(err, s.restore(ctx, spanCtx, released))
		}
	}

	return s.swap(ctx, targets, fresh), nil
}

func (s *Server) bindFresh(ctx, spanCtx context.Context, target Target, fresh map[listenerKey]*portListener) error {
	portListener, err := s.listen(ctx, spanCtx, target)
	if err != nil {
		return err
	}

	fresh[target.key()] = portListener

	return nil
}

func (s *Server) swap(ctx context.Context, targets []Target, fresh map[listenerKey]*portListener) <-chan struct{} {
	current := newRequests()
	wanted := make(map[listenerKey]struct{}, len(targets))

	for _, target := range targets {
		wanted[target.key()] = struct{}{}

		portListener, ok := fresh[target.key()]
		if !ok {
			portListener = s.listeners[target.key()]
		}

		portListener.route.Store(&route{handler: target.Handler, requests: current})
	}

	for key, portListener := range s.listeners {
		if _, ok := wanted[key]; !ok {
			s.retireListener(ctx, portListener)
		}
	}

	for key, portListener := range fresh {
		s.listeners[key] = portListener
		s.serve(ctx, portListener)
	}

	previous := s.requests
	s.requests = current

	if previous == nil {
		return nil
	}

	return s.retire(ctx, previous)
}

func (s *Server) restore(ctx, spanCtx context.Context, released []*portListener) error {
	var errs []error

	for _, previous := range released {
		portListener, err := s.listen(ctx, spanCtx, previous.target)
		if err != nil {
			errs = append(errs, err)

			continue
		}

		portListener.route.Store(previous.route.Load())
		s.listeners[previous.target.key()] = portListener
		s.serve(ctx, portListener)
	}

	return errors.Join(errs...)
}

func (s *Server) serve(ctx context.Context, portListener *portListener) {
	s.wg.Go(func() {
		err := portListener.Serve(portListener.listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
			telemetry.Error(ctx, "listener stopped", telemetry.Err(err))
		}
	})
}

func (s *Server) listen(ctx, spanCtx context.Context, target Target) (*portListener, error) {
	_, span := telemetry.Start(spanCtx, "server.listen")
	defer span.End()

	span.SetAttributes(
		semconv.ServerAddress(target.Address),
		attribute.Bool("uncors.tls", target.EnableTLS),
	)

	listener, err := s.bind(ctx, target)
	if err != nil {
		telemetry.Fail(span, err)

		return nil, err
	}

	portCtx, portCtxCancel := context.WithCancel(context.WithoutCancel(ctx))

	portListener := &portListener{
		target:   target,
		listener: listener,
		cancel:   portCtxCancel,
	}
	portListener.BaseContext = func(_ net.Listener) context.Context {
		return portCtx
	}
	portListener.ConnContext = withConn
	portListener.ReadHeaderTimeout = readHeaderTimeout
	portListener.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		s.dispatch(portListener, writer, request)
	})

	return portListener, nil
}

func withConn(ctx context.Context, conn net.Conn) context.Context {
	if tlsConn, ok := conn.(*tls.Conn); ok {
		conn = tlsConn.NetConn()
	}

	return context.WithValue(ctx, connKey{}, conn)
}

func (s *Server) dispatch(portListener *portListener, writer http.ResponseWriter, request *http.Request) {
	conn, _ := request.Context().Value(connKey{}).(net.Conn)

	current := portListener.route.Load()
	for !current.requests.acquire(conn) {
		next := portListener.route.Load()
		if next == current {
			s.handleRequest(unavailable, nil, writer, request)

			return
		}

		current = next
	}

	defer current.requests.release(conn)

	s.handleRequest(current.handler, current.requests, writer, request)
}

var unavailable = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
	infra.WriteCorsHeaders(writer.Header(), request.Header.Get("Origin"))
	writer.Header().Set("Connection", "close")
	writer.WriteHeader(http.StatusServiceUnavailable)
})

func (s *Server) retireListener(ctx context.Context, portListener *portListener) {
	delete(s.listeners, portListener.target.key())
	_ = portListener.closeListener()
	s.closing[portListener] = struct{}{}

	ctx = context.WithoutCancel(ctx)

	s.wg.Go(func() {
		shutdownCtx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
		defer cancel()

		err := portListener.drain(shutdownCtx)
		if err != nil {
			_ = portListener.Close()
		}

		portListener.cancel()

		s.mu.Lock()
		delete(s.closing, portListener)
		s.mu.Unlock()
	})
}

func (s *Server) retire(ctx context.Context, previous *requests) <-chan struct{} {
	retired := make(chan struct{})
	idle := previous.retire()
	s.retiring[previous] = struct{}{}

	ctx = context.WithoutCancel(ctx)

	s.wg.Go(func() {
		defer close(retired)

		timer := time.NewTimer(s.shutdownTimeout)
		defer timer.Stop()

		select {
		case <-idle:
		case <-previous.aborted:
		case <-timer.C:
			s.closeInFlight(ctx, previous)
		}

		s.mu.Lock()
		delete(s.retiring, previous)
		s.mu.Unlock()
	})

	return retired
}

func (s *Server) shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
	defer cancel()

	listeners, retiring, current := s.release()

	var (
		waitGroup sync.WaitGroup
		errsMu    sync.Mutex
		errs      []error
	)

	for _, server := range listeners {
		waitGroup.Add(1)
		go func(srv *portListener) {
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
	abort(retiring)

	if current != nil {
		errs = append(errs, current.drain(ctx))
	}

	return errors.Join(errs...)
}

func (s *Server) release() ([]*portListener, []*requests, *requests) {
	s.mu.Lock()
	defer s.mu.Unlock()

	listeners := make([]*portListener, 0, len(s.listeners)+len(s.closing))
	for _, portListener := range s.listeners {
		listeners = append(listeners, portListener)
	}

	for portListener := range s.closing {
		listeners = append(listeners, portListener)
	}

	retiring := make([]*requests, 0, len(s.retiring))
	for previous := range s.retiring {
		retiring = append(retiring, previous)
	}

	current := s.requests

	clear(s.listeners)
	s.requests = nil

	return listeners, retiring, current
}

func abort(retiring []*requests) {
	for _, previous := range retiring {
		previous.abort()
	}
}

func closeListeners(listeners map[listenerKey]*portListener) {
	for _, portListener := range listeners {
		_ = portListener.closeListener()
	}
}

func (s *Server) closeInFlight(ctx context.Context, previous *requests) {
	ctx, span := telemetry.Start(ctx, "server.close_in_flight")
	defer span.End()

	span.AddEvent("uncors.inflight_closed")
	telemetry.Warn(ctx, "in-flight requests did not finish before restart, closing them")
	s.publisher.Publish(events.Notice{
		Level: events.NoticeWarning,
		Text:  "In-flight requests did not finish before restart and were closed",
	})

	err := previous.closeConns()
	if err != nil {
		telemetry.Fail(span, err)
		s.publisher.Publish(events.Notice{
			Level: events.NoticeError,
			Text:  "Failed to close connections before restart: " + err.Error(),
		})
	}
}

func (s *Server) handleRequest(
	handler http.Handler,
	tracked *requests,
	writer http.ResponseWriter,
	request *http.Request,
) {
	helpers.NormaliseRequest(request)

	requestID := s.nextID.Add(1)
	rec := NewResponseRecorder(writer)
	rec.requests = tracked

	// The span is started before the scope so the scope's context carries it.
	ctx, span := telemetry.StartServer(request.Context(), "HTTP")
	defer span.End()

	if span.IsRecording() {
		annotateRequest(span, request, requestID)
		rec.Header().Set(telemetry.TraceIDHeader, span.SpanContext().TraceID().String())
	}

	telemetry.RequestStarted(ctx)

	ctx, scope := tracking.Begin(ctx, s.publisher, requestID, request, rec.startedAt)
	request = request.WithContext(ctx)

	defer func() {
		abort := recoverHandler(rec, request, recover())

		finishRequest(span, scope, rec, request)
		rec.releaseHijacked()

		if abort {
			panic(http.ErrAbortHandler)
		}
	}()

	handler.ServeHTTP(rec, request)
}

func recoverHandler(rec *ResponseRecorder, request *http.Request, recovered any) bool {
	if recovered == nil {
		return false
	}

	recoveredErr, isErr := recovered.(error)
	if isErr && errors.Is(recoveredErr, http.ErrAbortHandler) {
		return true
	}

	err := fmt.Errorf("%w: %v", errHandlerPanicked, recovered)
	ctx := request.Context()

	if telemetry.Enabled() {
		telemetry.Error(ctx, "handler panicked",
			slog.String("method", request.Method),
			slog.String("url", helpers.LogURL(request.URL)),
			telemetry.Err(err),
			slog.String("stack", string(debug.Stack())),
		)
	}

	tracking.Fail(ctx, err)

	if ctx.Err() != nil || infra.ResponseStarted(rec) {
		return true
	}

	infra.HTTPError(rec, request, err)

	return false
}

func finishRequest(span trace.Span, scope *tracking.Scope, rec *ResponseRecorder, request *http.Request) {
	ctx := request.Context()

	data := &contracts.RequestData{
		Method:    request.Method,
		URL:       request.URL,
		Header:    request.Header,
		Code:      helpers.NormaliseStatusCode(rec.StatusCode()),
		Cancelled: ctx.Err() != nil,
	}

	if span.IsRecording() {
		annotateResponse(span, data, rec.Header(), rec.Written())
	}

	if telemetry.Enabled() {
		telemetry.RequestFinished(ctx, telemetry.RequestMetrics{
			Method:        request.Method,
			Handler:       scope.Handler().String(),
			Status:        data.Code,
			Cancelled:     data.Cancelled,
			Duration:      time.Since(rec.startedAt),
			ResponseBytes: rec.Written(),
		})
	}

	scope.Finish(data)
}
