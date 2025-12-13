package cache

import (
	"bytes"
	"io"
	"net/http"

	"github.com/go-http-utils/headers"
)

type LegacyCachedResponse struct {
	Code   int
	Body   []byte
	Header http.Header
}

type LegacyCacheableResponseWriter struct {
	original     http.ResponseWriter
	outputWriter io.Writer
	buffer       *bytes.Buffer
	code         int
}

func NewLegacyCacheableWriter(original http.ResponseWriter) *LegacyCacheableResponseWriter {
	buffer := &bytes.Buffer{}

	return &LegacyCacheableResponseWriter{
		original:     original,
		outputWriter: io.MultiWriter(buffer, original),
		buffer:       buffer,
	}
}

func (w *LegacyCacheableResponseWriter) Header() http.Header {
	return w.original.Header()
}

func (w *LegacyCacheableResponseWriter) Write(bytes []byte) (int, error) {
	return w.outputWriter.Write(bytes)
}

func (w *LegacyCacheableResponseWriter) StatusCode() int {
	return w.code
}

func (w *LegacyCacheableResponseWriter) WriteHeader(statusCode int) {
	w.code = statusCode
	w.original.WriteHeader(statusCode)
}

func (w *LegacyCacheableResponseWriter) GetCachedResponse() *LegacyCachedResponse {
	header := w.original.Header().Clone()
	cleanupHeader(header)

	return &LegacyCachedResponse{
		Code:   w.code,
		Body:   w.buffer.Bytes(),
		Header: header,
	}
}

func cleanupHeader(header http.Header) {
	header.Del(headers.ContentLength)
}
