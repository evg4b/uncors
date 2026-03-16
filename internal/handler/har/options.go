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
// are included in the HAR output. Defaults to false to prevent credentials
// from being persisted to disk.
//
// Filtered headers when false:
//   - Cookie / Set-Cookie  (session identifiers)
//   - Authorization        (Bearer tokens, Basic credentials)
//   - WWW-Authenticate     (server auth challenges)
//   - Proxy-Authorization  (proxy credentials)
//   - Proxy-Authenticate   (proxy auth challenges)
func WithCaptureSecureHeaders(capture bool) MiddlewareOption {
	return func(m *Middleware) {
		m.captureSecureHeaders = capture
	}
}
