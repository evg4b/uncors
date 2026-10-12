package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/tracking"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
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

	mu              sync.RWMutex
	listeners       []*PortListener
	manager         *HostCertManager
	publisher       events.Publisher
	nextID          atomic.Uint64
	shutdownTimeout time.Duration
}

func New(manager *HostCertManager, publisher events.Publisher) *Server {
	if publisher == nil {
		publisher = events.NoopPublisher{}
	}

	return &Server{
		listeners:       []*PortListener{},
		manager:         manager,
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
	s.Add(1)
	defer s.Done()

	return telemetry.Run(ctx, "server.restart", func(ctx context.Context) error {
		// Shutdown closes the listeners straight away and then waits for in-flight
		// requests. A long poll or a stream can outlive that wait; it is cut off
		// rather than left to keep the server without any listener.
		err := s.Shutdown(ctx)
		if err != nil {
			s.closeInFlight(ctx, err)
		}

		return s.Start(ctx, targets)
	})
}

func (s *Server) Wait() {
	s.WaitGroup.Wait()
}

func (s *Server) Close() error {
	s.mu.RLock()
	listeners := s.listeners
	s.mu.RUnlock()

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
func (s *Server) start(ctx, spanCtx context.Context, targets []Target) error {
	trace.SpanFromContext(spanCtx).SetAttributes(attribute.Int("uncors.listeners", len(targets)))

	s.mu.Lock()
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
	s.mu.Unlock()

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

			_, span := telemetry.Start(spanCtx, "server.listen")
			span.SetAttributes(
				semconv.ServerAddress(srv.target.Address),
				attribute.Bool("uncors.tls", srv.target.EnableTLS),
			)

			ready := func() {
				span.End()
				launchWaitGroup.Done()
			}

			err := srv.Listen(ctx, func() {
				once.Do(ready)
			})

			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				telemetry.Fail(span, err)
				launchErrorsMu.Lock()

				launchErrs = append(launchErrs, err)
				launchErrorsMu.Unlock()
			}
			// onReady is never called on bind failure; after it fired this is a no-op.
			once.Do(ready)
		}(server)
	}

	launchWaitGroup.Wait()

	launchErrorsMu.Lock()
	err := errors.Join(launchErrs...)
	launchErrorsMu.Unlock()

	return err
}

func (s *Server) shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
	defer cancel()

	s.mu.RLock()
	listeners := s.listeners
	s.mu.RUnlock()

	var (
		waitGroup sync.WaitGroup
		errsMu    sync.Mutex
		errs      []error
	)

	for _, server := range listeners {
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

func (s *Server) closeInFlight(ctx context.Context, shutdownErr error) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent("uncors.inflight_closed")
	telemetry.Warn(ctx, "in-flight requests did not finish before restart, closing them",
		slog.String("error", shutdownErr.Error()))
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

func (s *Server) handleRequest(handler contracts.Handler, writer http.ResponseWriter, request *http.Request) {
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

	ctx, scope := tracking.Begin(ctx, s.publisher, requestID, request)

	err := handler.ServeHTTP(rec, request.WithContext(ctx))
	if err != nil {
		tracking.Fail(ctx, err)
		infra.HTTPError(rec, err)
	}

	data := helpers.ToRequestData(request, helpers.NormaliseStatusCode(rec.StatusCode()))
	data.Cancelled = ctx.Err() != nil

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
