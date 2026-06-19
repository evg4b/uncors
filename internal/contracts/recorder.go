package contracts

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

// ResponseCapture holds the captured response data after a request completes.
type ResponseCapture struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	Duration   time.Duration
}

// BodyCapturer is implemented by ResponseRecorder; middleware use it to opt in
// to body buffering and read the captured response after calling next.
type BodyCapturer interface {
	EnableBodyCapture()
	Captured() ResponseCapture
}

// ResponseRecorder wraps http.ResponseWriter, optionally buffers the response
// body, and exposes the captured data via Captured() after the handler returns.
// It is the single capture point for HAR recording, caching, and request tracking.
type ResponseRecorder struct {
	http.ResponseWriter

	statusCode int
	buf        *bytes.Buffer
	output     io.Writer
	startedAt  time.Time
}

func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	r := &ResponseRecorder{
		ResponseWriter: w,
		startedAt:      time.Now(),
	}
	r.output = w

	return r
}

// WrapResponseWriter creates a ResponseRecorder around w.
// It satisfies ResponseWriter and BodyCapturer.
func WrapResponseWriter(w http.ResponseWriter) *ResponseRecorder {
	return NewResponseRecorder(w)
}

func (r *ResponseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *ResponseRecorder) Write(b []byte) (int, error) {
	return r.output.Write(b)
}

func (r *ResponseRecorder) StatusCode() int {
	return r.statusCode
}

// EnableBodyCapture turns on response body buffering. Must be called before the
// handler writes any body bytes. Subsequent calls are no-ops.
func (r *ResponseRecorder) EnableBodyCapture() {
	if r.buf != nil {
		return
	}

	r.buf = &bytes.Buffer{}
	r.output = io.MultiWriter(r.buf, r.ResponseWriter)
}

// Captured returns a snapshot of the response as written so far.
// Body is nil unless EnableBodyCapture was called before the handler ran.
func (r *ResponseRecorder) Captured() ResponseCapture {
	var body []byte
	if r.buf != nil {
		body = r.buf.Bytes()
	}

	return ResponseCapture{
		StatusCode: normaliseStatusCode(r.statusCode),
		Header:     r.Header(),
		Body:       body,
		Duration:   time.Since(r.startedAt),
	}
}

func normaliseStatusCode(code int) int {
	if code == 0 {
		return http.StatusOK
	}

	return code
}
