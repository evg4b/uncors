package handler

import (
	"sync"

	"github.com/evg4b/uncors/internal/contracts"
)

// LazyHandler wraps an init function and defers handler creation to the first
// ServeHTTP call. Subsequent calls reuse the same handler instance.
func LazyHandler(factory func() contracts.Handler) contracts.Handler {
	var (
		once    sync.Once
		handler contracts.Handler
	)

	return contracts.HandlerFunc(func(w contracts.ResponseWriter, r *contracts.Request) {
		once.Do(func() {
			handler = factory()
		})
		handler.ServeHTTP(w, r)
	})
}

type lazyMiddleware struct {
	sync.Once

	factory func() contracts.Middleware
	wrapped contracts.Handler
}

// LazyMiddleware wraps an init function and defers middleware creation to the
// first ServeHTTP call of the returned handler. Each Wrap call produces an
// independent lazy instance with its own sync.Once.
func LazyMiddleware(factory func() contracts.Middleware) contracts.Middleware {
	return &lazyMiddleware{
		factory: factory,
	}
}

func (l *lazyMiddleware) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(w contracts.ResponseWriter, r *contracts.Request) {
		l.Do(func() {
			l.wrapped = l.factory().Wrap(next)
		})
		l.wrapped.ServeHTTP(w, r)
	})
}
