package cache

import (
	"bytes"
	"io"
	"net/http"

	"github.com/go-http-utils/headers"
)

type CachedResponse struct {
	Code   int
	Body   []byte
	Header http.Header
}

type CacheableResponseWriter struct {
	original     http.ResponseWriter
	outputWriter io.Writer
	buffer       *bytes.Buffer
	code         int
}

func NewCacheableWriter(original http.ResponseWriter) *CacheableResponseWriter {
	buffer := &bytes.Buffer{}

	return &CacheableResponseWriter{
		original:     original,
		outputWriter: io.MultiWriter(buffer, original),
		buffer:       buffer,
	}
}

func (w *CacheableResponseWriter) Header() http.Header {
	return w.original.Header()
}

func (w *CacheableResponseWriter) Write(bytes []byte) (int, error) {
	return w.outputWriter.Write(bytes) //nolint: wrapcheck
}

func (w *CacheableResponseWriter) StatusCode() int {
	return w.code
}

func (w *CacheableResponseWriter) WriteHeader(statusCode int) {
	w.code = statusCode
	w.original.WriteHeader(statusCode)
}

func (w *CacheableResponseWriter) GetCachedResponse() *CachedResponse {
	header := w.original.Header().Clone()
	header.Del(headers.ContentLength)

	return &CachedResponse{
		Code:   w.code,
		Body:   w.buffer.Bytes(),
		Header: header,
	}
}
