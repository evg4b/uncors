package infra

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/go-http-utils/headers"
)

// DefaultCaptureLimit bounds how much of a response body is kept in memory for
// recording or caching. Without a bound, proxying a large download through a
// mapping with HAR or cache enabled would buffer the whole thing.
const DefaultCaptureLimit = 5 << 20 // 5 MiB

// ResponseRecorder wraps the response writer of a request to observe its status
// code and, on request, its body.
//
// It forwards the optional interfaces net/http's own writer implements. A
// wrapper that swallows them silently breaks streaming: server sent events would
// sit in the transport buffer, and connection upgrades would be impossible.
type ResponseRecorder struct {
	http.ResponseWriter

	statusCode int
	buf        *bytes.Buffer
	limit      int64
	truncated  bool
	hijacked   bool
	startedAt  time.Time
}

// CaptureFrom returns the body capturer installed for this request, unwrapping
// any writers layered on top of it. Middleware that needs the response body asks
// for the existing capturer instead of assuming one is there, so a writer it did
// not install can never turn a request into a panic.
func CaptureFrom(writer http.ResponseWriter) (contracts.BodyCapturer, bool) {
	for {
		if capturer, ok := writer.(contracts.BodyCapturer); ok {
			return capturer, true
		}

		unwrapper, ok := writer.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return nil, false
		}

		writer = unwrapper.Unwrap()
	}
}

func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	return &ResponseRecorder{
		ResponseWriter: w,
		startedAt:      time.Now(),
	}
}

// Unwrap exposes the underlying writer, which is the convention net/http uses to
// look through response writer wrappers (http.NewResponseController).
func (r *ResponseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func (r *ResponseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode

	if r.buf != nil && r.isStreaming() {
		// A streamed response has no meaningful "body" to hold in memory, and
		// buffering one would delay every event.
		r.buf = nil
	}

	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *ResponseRecorder) Write(chunk []byte) (int, error) {
	r.capture(chunk)

	return r.ResponseWriter.Write(chunk)
}

// ReadFrom lets io.Copy use the underlying writer's fast path when nothing needs
// the body captured.
func (r *ResponseRecorder) ReadFrom(src io.Reader) (int64, error) {
	readerFrom, ok := r.ResponseWriter.(io.ReaderFrom)
	if !ok || r.buf != nil {
		return io.Copy(writerOnly{r}, src)
	}

	return readerFrom.ReadFrom(src)
}

// Flush forwards to the underlying writer so that streamed responses reach the
// client as they are produced.
func (r *ResponseRecorder) Flush() {
	flusher, ok := r.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

// Hijack forwards connection upgrades (WebSocket and friends). A hijacked
// connection is no longer an HTTP response, so nothing is recorded for it.
func (r *ResponseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}

	conn, readWriter, err := hijacker.Hijack()
	if err == nil {
		r.hijacked = true
		r.buf = nil
	}

	return conn, readWriter, err //nolint:wrapcheck // forwarding the writer's own error
}

func (r *ResponseRecorder) StatusCode() int {
	return r.statusCode
}

// EnableBodyCapture starts capturing, keeping at most maxBytes. When capture is
// already on, the larger of the two limits wins: consumers apply their own
// policy to the result, and one of them asking for less must not shorten what
// another one sees.
func (r *ResponseRecorder) EnableBodyCapture(maxBytes int64) {
	if r.hijacked {
		return
	}

	if maxBytes <= 0 {
		maxBytes = DefaultCaptureLimit
	}

	r.limit = max(r.limit, maxBytes)

	if r.buf == nil {
		r.buf = &bytes.Buffer{}
	}
}

func (r *ResponseRecorder) Captured() contracts.ResponseCapture {
	var body []byte
	if r.buf != nil {
		body = r.buf.Bytes()
	}

	return contracts.ResponseCapture{
		StatusCode: normaliseStatusCode(r.statusCode),
		Header:     r.Header(),
		Body:       body,
		Truncated:  r.truncated,
		Duration:   time.Since(r.startedAt),
	}
}

func (r *ResponseRecorder) capture(chunk []byte) {
	if r.buf == nil {
		return
	}

	remaining := r.limit - int64(r.buf.Len())
	if remaining <= 0 {
		r.truncated = true

		return
	}

	if int64(len(chunk)) > remaining {
		r.buf.Write(chunk[:remaining])
		r.truncated = true

		return
	}

	r.buf.Write(chunk)
}

// isStreaming reports whether the response announces itself as a stream, in
// which case capturing it would buffer an endless body.
func (r *ResponseRecorder) isStreaming() bool {
	header := r.Header()

	if strings.HasPrefix(header.Get(headers.ContentType), "text/event-stream") {
		return true
	}

	return header.Get(headers.ContentLength) == "" && r.statusCode == http.StatusSwitchingProtocols
}

func normaliseStatusCode(code int) int {
	if code == 0 {
		return http.StatusOK
	}

	return code
}

// writerOnly hides ReadFrom so that io.Copy falls back to Write, which is what
// keeps body capture working on the copy path.
type writerOnly struct {
	io.Writer
}
