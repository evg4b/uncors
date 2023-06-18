package contracts

import (
	"net/http"
)

type ResponseWriter struct {
	StatusCode int
	http.ResponseWriter
}

func WrapResponseWriter(writer http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{ResponseWriter: writer}
}

func (r *ResponseWriter) WriteHeader(statusCode int) {
	r.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
