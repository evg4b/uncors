package har

import (
	"bytes"
	"io"
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
)

// captureWriter wraps a ResponseWriter and tees the response body
// into an internal buffer so the middleware can build a HAR entry
// after the handler returns. It satisfies contracts.ResponseWriter.
type captureWriter struct {
	contracts.ResponseWriter

	buffer bytes.Buffer
	output io.Writer
	code   int
}

func newCaptureWriter(w contracts.ResponseWriter) *captureWriter {
	cw := &captureWriter{
		ResponseWriter: w,
		code:           http.StatusOK,
	}

	cw.output = io.MultiWriter(&cw.buffer, w)

	return cw
}

func (cw *captureWriter) Write(b []byte) (int, error) {
	return cw.output.Write(b)
}

func (cw *captureWriter) WriteHeader(statusCode int) {
	cw.code = statusCode
	cw.ResponseWriter.WriteHeader(statusCode)
}

func (cw *captureWriter) StatusCode() int {
	return cw.code
}

func (cw *captureWriter) body() []byte {
	return cw.buffer.Bytes()
}
