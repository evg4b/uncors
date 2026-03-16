package har

// MiddlewareOption is a functional option for Middleware.
type MiddlewareOption = func(*Middleware)

// WithWriter sets the HAR writer the middleware uses to persist entries.
func WithWriter(w *Writer) MiddlewareOption {
	return func(m *Middleware) {
		m.writer = w
	}
}
