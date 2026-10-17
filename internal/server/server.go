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

type Server struct {
	wg sync.WaitGroup

	mu              sync.RWMutex
	listeners       []*portListener
	targets         []Target
	getCertificate  func(*tls.ClientHelloInfo) (*tls.Certificate, error)
	publisher       events.Publisher
	nextID          atomic.Uint64
	shutdownTimeout time.Duration
}

func New(getCertificate func(*tls.ClientHelloInfo) (*tls.Certificate, error), publisher events.Publisher) *Server {
	return &Server{
		getCertificate:  getCertificate,
		publisher:       publisher,
		shutdownTimeout: shutdownTimeout,
	}
}

func (s *Server) Start(ctx context.Context, targets []Target) error {
	return telemetry.Run(ctx, "server.start", func(spanCtx context.Context) error {
		return s.start(ctx, spanCtx, targets)
	})
}

func (s *Server) Shutdown(ctx context.Context) error {
	return telemetry.Run(ctx, "server.shutdown", s.shutdown)
}

func (s *Server) Restart(ctx context.Context, targets []Target) error {
	s.wg.Add(1)
	defer s.wg.Done()

	return telemetry.Run(ctx, "server.restart", func(ctx context.Context) error {
		s.mu.RLock()
		previous := s.targets
		s.mu.RUnlock()

		// Shutdown closes the listeners straight away and then waits for in-flight
		// requests. A long poll or a stream can outlive that wait; it is cut off
		// rather than left to keep the server without any listener.
		err := s.Shutdown(ctx)
		if err != nil {
			s.closeInFlight(ctx, err)
		}

		err = s.Start(ctx, targets)
		if err != nil && len(previous) > 0 {
			// The engine keeps the previous generation when a restart fails, so
			// it is served again rather than leaving the proxy down.
			err = errors.Join(err, s.Start(ctx, previous))
		}

		return err
	})
}

func (s *Server) Wait() {
	s.wg.Wait()
}

func (s *Server) Close() error {
	listeners := s.release()

	var errs []error

	for _, portListener := range listeners {
		err := portListener.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// ctx is the listeners' base context. spanCtx carries the server.start span,
// which only the listen spans nest under: TLS handshakes and requests derive
// their contexts from the base one.
//
// Every target is bound before any is served, so a start that fails leaves
// nothing listening.
func (s *Server) start(ctx, spanCtx context.Context, targets []Target) error {
	trace.SpanFromContext(spanCtx).SetAttributes(attribute.Int("uncors.listeners", len(targets)))

	listeners := make([]*portListener, 0, len(targets))

	var errs []error

	for _, target := range targets {
		listener, err := s.listen(ctx, spanCtx, target)
		if err != nil {
			errs = append(errs, err)

			continue
		}

		listeners = append(listeners, listener)
	}

	if len(errs) > 0 {
		for _, listener := range listeners {
			_ = listener.closeListener()
		}

		return errors.Join(errs...)
	}

	s.mu.Lock()
	s.listeners = listeners
	s.targets = targets
	s.mu.Unlock()

	for _, listener := range listeners {
		s.wg.Go(func() {
			err := listener.Serve(listener.listener)
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				telemetry.Error(ctx, "listener stopped", telemetry.Err(err))
			}
		})
	}

	return nil
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

	portCtx, portCtxCancel := context.WithCancel(ctx)

	portListener := &portListener{
		Server: http.Server{
			BaseContext: func(_ net.Listener) context.Context {
				return portCtx
			},
			ReadHeaderTimeout: readHeaderTimeout,
			Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				s.handleRequest(target.Handler, writer, request)
			}),
		},
		listener: listener,
	}

	portListener.RegisterOnShutdown(portCtxCancel)

	return portListener, nil
}

func (s *Server) shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
	defer cancel()

	listeners := s.release()

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

	return errors.Join(errs...)
}

// release also forgets the targets: a stopped server has nothing to fall back to.
func (s *Server) release() []*portListener {
	s.mu.Lock()
	defer s.mu.Unlock()

	listeners := s.listeners
	s.targets = nil

	return listeners
}

func (s *Server) closeInFlight(ctx context.Context, shutdownErr error) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent("uncors.inflight_closed")
	telemetry.Warn(ctx, "in-flight requests did not finish before restart, closing them",
		telemetry.Err(shutdownErr))
	s.publisher.Publish(events.Notice{
		Level: events.NoticeWarning,
		Text:  "In-flight requests did not finish before restart and were closed",
	})

	err := s.Close()
	if err != nil {
		telemetry.Fail(span, err)
		s.publisher.Publish(events.Notice{
			Level: events.NoticeError,
			Text:  "Failed to close connections before restart: " + err.Error(),
		})
	}
}

func (s *Server) handleRequest(handler http.Handler, writer http.ResponseWriter, request *http.Request) {
	helpers.NormaliseRequest(request)

	requestID := s.nextID.Add(1)
	rec := NewResponseRecorder(writer)

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
