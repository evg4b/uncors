package cache

import (
	"bytes"
	"io"
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/samber/lo"
)

type CacheableResponseWriter struct {
	http.ResponseWriter

	storage contracts.Cache
	key     string
	output  io.Writer
	buffer  bytes.Buffer
	code    int
}

func NewCacheableResponseWriter(
	cache contracts.Cache,
	original http.ResponseWriter,
	key string,
) *CacheableResponseWriter {
	writer := &CacheableResponseWriter{
		ResponseWriter: original,
		storage:        cache,
		key:            key,
		buffer:         bytes.Buffer{},
		code:           http.StatusOK,
	}

	writer.output = io.MultiWriter(
		&writer.buffer,
		original,
	)

	return writer
}

func (w *CacheableResponseWriter) Write(bytes []byte) (int, error) {
	return w.output.Write(bytes)
}

func (w *CacheableResponseWriter) StatusCode() int {
	return w.code
}

func (w *CacheableResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.code = statusCode
}

func (w *CacheableResponseWriter) Close() {
	if !helpers.Is2xxCode(w.code) {
		return
	}

	w.storage.Set(w.key, contracts.CachedResponse{
		Code: w.code,
		Body: w.buffer.Bytes(),
		Headers: lo.MapToSlice(w.Header(), func(key string, value []string) contracts.CachedHeader {
			return contracts.CachedHeader{
				Name:  key,
				Value: value,
			}
		}),
	})
}
