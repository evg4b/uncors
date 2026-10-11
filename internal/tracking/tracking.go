package tracking

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type (
	scopeKey    struct{}
	rootSpanKey struct{}
)

type Scope struct {
	publisher events.Publisher
	id        uint64
	method    string
	url       *url.URL
	startedAt time.Time

	mu       sync.Mutex
	started  bool
	finished bool
	handler  events.HandlerKind
	rewrite  *events.Rewrite
	err      error
}

func Begin(
	ctx context.Context,
	publisher events.Publisher,
	id uint64,
	request *contracts.Request,
) (context.Context, *Scope) {
	scope := &Scope{
		publisher: publisher,
		id:        id,
		method:    request.Method,
		url:       request.URL,
		startedAt: time.Now(),
	}

	// Attribution is recorded on the request's root span. It is kept apart
	// from the scope so tracing adds nothing to a request when it is off.
	if telemetry.Enabled() {
		ctx = context.WithValue(ctx, rootSpanKey{}, trace.SpanFromContext(ctx))
	}

	return context.WithValue(ctx, scopeKey{}, scope), scope
}

func Claim(ctx context.Context, kind events.HandlerKind) {
	if scope := fromContext(ctx); scope != nil {
		scope.claim(kind)

		if root := rootSpan(ctx); root != nil {
			root.SetAttributes(attribute.String("uncors.handler", kind.String()))
			trace.SpanFromContext(ctx).AddEvent("claimed")
		}
	}
}

func Rewrote(ctx context.Context, from, to string) {
	if scope := fromContext(ctx); scope != nil {
		scope.mu.Lock()
		defer scope.mu.Unlock()

		scope.rewrite = &events.Rewrite{From: from, To: to}

		if root := rootSpan(ctx); root != nil {
			root.SetAttributes(
				attribute.String("uncors.rewrite.from", from),
				attribute.String("uncors.rewrite.to", to),
			)
		}
	}
}

func Fail(ctx context.Context, err error) {
	if scope := fromContext(ctx); scope != nil {
		scope.mu.Lock()
		defer scope.mu.Unlock()

		scope.err = err

		if root := rootSpan(ctx); root != nil {
			telemetry.Fail(root, err)
		}
	}
}

func (s *Scope) Finish(data *contracts.RequestData) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.finished {
		return
	}

	s.finished = true
	s.publishStarted()
	s.publisher.Publish(events.RequestCompleted{
		ID:       s.id,
		Handler:  s.handler,
		Rewrite:  s.rewrite,
		Data:     data,
		Duration: time.Since(s.startedAt),
		Err:      s.err,
	})
}

func (s *Scope) claim(kind events.HandlerKind) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.finished {
		return
	}

	s.handler = kind
	s.publishStarted()
}

// publishStarted runs under mu, which orders RequestStarted before
// RequestCompleted for every subscriber.
func (s *Scope) publishStarted() {
	if s.started {
		return
	}

	s.started = true
	s.publisher.Publish(events.RequestStarted{
		ID:      s.id,
		Method:  s.method,
		URL:     s.url,
		Handler: s.handler,
		Rewrite: s.rewrite,
		At:      s.startedAt,
	})
}

// rootSpan returns nil unless tracing is on and the span is recording.
func rootSpan(ctx context.Context) trace.Span {
	if !telemetry.Enabled() {
		return nil
	}

	span, _ := ctx.Value(rootSpanKey{}).(trace.Span)
	if span == nil || !span.IsRecording() {
		return nil
	}

	return span
}

func fromContext(ctx context.Context) *Scope {
	scope, _ := ctx.Value(scopeKey{}).(*Scope)

	return scope
}
