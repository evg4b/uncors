package server

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
)

type ResponseRecorder struct {
	http.ResponseWriter

	statusCode int
	buf        *bytes.Buffer
	output     io.Writer
	startedAt  time.Time
}

func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	rec := &ResponseRecorder{
		ResponseWriter: w,
		startedAt:      time.Now(),
	}
	rec.output = w

	return rec
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

func (r *ResponseRecorder) EnableBodyCapture() {
	if r.buf != nil {
		return
	}

	r.buf = &bytes.Buffer{}
	r.output = io.MultiWriter(r.buf, r.ResponseWriter)
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
		Duration:   time.Since(r.startedAt),
	}
}

func normaliseStatusCode(code int) int {
	if code == 0 {
		return http.StatusOK
	}

	return code
}
