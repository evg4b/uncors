package har

// MiddlewareOption is a functional option for Middleware.
type MiddlewareOption = func(*Middleware)

// WithWriter sets the HAR writer the middleware uses to persist entries.
func WithWriter(w *Writer) MiddlewareOption {
	return func(m *Middleware) {
		m.writer = w
	}
}

// WithCaptureCookies controls whether Cookie request headers and Set-Cookie
// response headers are included in the HAR output. Defaults to false to
// avoid accidentally persisting sensitive session data.
func WithCaptureCookies(capture bool) MiddlewareOption {
	return func(m *Middleware) {
		m.captureCookies = capture
	}
}
