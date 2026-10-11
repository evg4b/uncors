package har

type MiddlewareOption = func(*Middleware)

func WithWriter(w *Writer) MiddlewareOption {
	return func(m *Middleware) {
		m.writer = w
	}
}

func WithCaptureSecureHeaders(capture bool) MiddlewareOption {
	return func(m *Middleware) {
		m.captureSecureHeaders = capture
	}
}
