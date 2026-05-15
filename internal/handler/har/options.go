package har

// MiddlewareOption is a functional option for Middleware.
type MiddlewareOption = func(*Middleware)

// WithWriter sets the HAR writer the middleware uses to persist entries.
func WithWriter(w *Writer) MiddlewareOption {
	return func(m *Middleware) {
		m.writer = w
	}
}

// WithCaptureSecureHeaders controls whether security-sensitive HTTP headers
// (cookies, Authorization, WWW-Authenticate, Proxy-Authorization) are included
// in the HAR output. Defaults to false.
func WithCaptureSecureHeaders(capture bool) MiddlewareOption {
	return func(m *Middleware) {
		m.captureSecureHeaders = capture
	}
}
