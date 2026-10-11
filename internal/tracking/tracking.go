// Package tracking attributes each request to the component that produced its
// response and reports the request's lifecycle as two events: RequestStarted
// and RequestCompleted.
//
// The server opens a Scope per request. Components report through the request
// context: Claim when they decide to produce the response, Rewrote when they
// change the path, Fail when the handler chain ended in an error. Only the
// component that answers claims, so a cache miss is reported by the proxy that
// fetched the response and a cache hit by the cache.
package tracking

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
)

type scopeKey struct{}

// Scope tracks one request. Handlers may report from goroutines of their own,
// so every method is safe for concurrent use.
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

// Begin opens a scope for request and returns a context that carries it.
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

	return context.WithValue(ctx, scopeKey{}, scope), scope
}

// Claim records that kind produces the response for the request in ctx. The
// first claim publishes RequestStarted; a later one replaces the handler that
// RequestCompleted reports.
func Claim(ctx context.Context, kind events.HandlerKind) {
	if scope := fromContext(ctx); scope != nil {
		scope.claim(kind)
	}
}

// Rewrote records that the request path was rewritten from to to.
func Rewrote(ctx context.Context, from, to string) {
	if scope := fromContext(ctx); scope != nil {
		scope.mu.Lock()
		defer scope.mu.Unlock()

		scope.rewrite = &events.Rewrite{From: from, To: to}
	}
}

// Fail records the error the handler chain ended with.
func Fail(ctx context.Context, err error) {
	if scope := fromContext(ctx); scope != nil {
		scope.mu.Lock()
		defer scope.mu.Unlock()

		scope.err = err
	}
}

// Finish publishes RequestCompleted, preceded by RequestStarted when no
// component claimed the request. Calls after the first are ignored.
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

func fromContext(ctx context.Context) *Scope {
	scope, _ := ctx.Value(scopeKey{}).(*Scope)

	return scope
}
