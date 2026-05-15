package har

import (
	"bytes"
	"io"
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
)

type captureWriter struct {
	contracts.ResponseWriter

	buffer bytes.Buffer
	output io.Writer
	code   int
}

func newCaptureWriter(wrapped contracts.ResponseWriter) *captureWriter {
	capture := &captureWriter{
		ResponseWriter: wrapped,
		code:           http.StatusOK,
	}

	capture.output = io.MultiWriter(&capture.buffer, wrapped)

	return capture
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
