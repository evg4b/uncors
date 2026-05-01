package handler

import (
	"sync"

	"github.com/evg4b/uncors/internal/contracts"
)

// LazyHandler wraps an init function and defers handler creation to the first
// ServeHTTP call. Subsequent calls reuse the same handler instance.
func LazyHandler(init func() contracts.Handler) contracts.Handler {
	var (
		once    sync.Once
		handler contracts.Handler
	)

	return contracts.HandlerFunc(func(w contracts.ResponseWriter, r *contracts.Request) {
		once.Do(func() {
			handler = init()
		})
		handler.ServeHTTP(w, r)
	})
}

type lazyMiddleware struct {
	sync.Once

	init    func() contracts.Middleware
	wrapped contracts.Handler
}

// LazyMiddleware wraps an init function and defers middleware creation to the
// first ServeHTTP call of the returned handler. Each Wrap call produces an
// independent lazy instance with its own sync.Once.
func LazyMiddleware(init func() contracts.Middleware) contracts.Middleware {
	return &lazyMiddleware{
		init: init,
	}
}

func (l *lazyMiddleware) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(w contracts.ResponseWriter, r *contracts.Request) {
		l.Do(func() {
			l.wrapped = l.init().Wrap(next)
		})
		l.wrapped.ServeHTTP(w, r)
	})
}
