package server

import (
	"bytes"
	"net/http"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
)

const maxCapturedBody = 10 << 20

type ResponseRecorder struct {
	http.ResponseWriter

	statusCode int
	buf        *bytes.Buffer
	truncated  bool
	startedAt  time.Time
	written    int64
}

func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	return &ResponseRecorder{
		ResponseWriter: w,
		startedAt:      time.Now(),
	}
}

func (r *ResponseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *ResponseRecorder) Write(b []byte) (int, error) {
	written, err := r.ResponseWriter.Write(b)
	r.written += int64(written)

	if r.buf != nil {
		r.capture(b[:written])
	}

	return written, err
}

func (r *ResponseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func (r *ResponseRecorder) Flush() {
	_ = r.FlushError()
}

func (r *ResponseRecorder) FlushError() error {
	return http.NewResponseController(r.ResponseWriter).Flush()
}

// Written is the number of body bytes sent to the client.
func (r *ResponseRecorder) Written() int64 {
	return r.written
}

func (r *ResponseRecorder) StatusCode() int {
	return r.statusCode
}

func (r *ResponseRecorder) EnableBodyCapture() {
	if r.buf != nil {
		return
	}

	r.buf = &bytes.Buffer{}
}

func (r *ResponseRecorder) Captured() contracts.ResponseCapture {
	var body []byte
	if r.buf != nil {
		body = r.buf.Bytes()
	}

	return contracts.ResponseCapture{
		StatusCode: helpers.NormaliseStatusCode(r.statusCode),
		Header:     r.Header(),
		Body:       body,
		Truncated:  r.truncated,
		Duration:   time.Since(r.startedAt),
	}
}

func (r *ResponseRecorder) capture(chunk []byte) {
	if r.truncated {
		return
	}

	if free := maxCapturedBody - r.buf.Len(); len(chunk) > free {
		r.buf.Write(chunk[:free])
		r.truncated = true

		return
	}

	r.buf.Write(chunk)
}
