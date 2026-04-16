package contracts

import "sync"

// LazyHandler wraps an init function and defers handler creation to the first
// ServeHTTP call. Subsequent calls reuse the same handler instance.
func LazyHandler(init func() Handler) Handler {
	var (
		once    sync.Once
		handler Handler
	)

	return HandlerFunc(func(w ResponseWriter, r *Request) {
		once.Do(func() { handler = init() })
		handler.ServeHTTP(w, r)
	})
}

type lazyMiddleware struct {
	init func() Middleware
}

// LazyMiddleware wraps an init function and defers middleware creation to the
// first ServeHTTP call of the returned handler. Each Wrap call produces an
// independent lazy instance with its own sync.Once.
func LazyMiddleware(init func() Middleware) Middleware {
	return &lazyMiddleware{init: init}
}

func (l *lazyMiddleware) Wrap(next Handler) Handler {
	var (
		once    sync.Once
		wrapped Handler
	)

	return HandlerFunc(func(w ResponseWriter, r *Request) {
		once.Do(func() { wrapped = l.init().Wrap(next) })
		wrapped.ServeHTTP(w, r)
	})
}
