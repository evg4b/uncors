package tui

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/contracts"
)

type RequestTracker struct{}

func (r RequestTracker) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) {
		log.Infof("Request: %s %s", request.Method, request.URL)
		w := NewResponseWriter(writer)
		next.ServeHTTP(w, request)
		log.Infof("Request: %s %s: %d", request.Method, request.URL, w.StatusCode())
	})
}

type ResponseWriter struct {
	writer contracts.ResponseWriter
}

func NewResponseWriter(writer contracts.ResponseWriter) ResponseWriter {
	return ResponseWriter{writer: writer}
}

func (r ResponseWriter) Header() http.Header {
	return r.writer.Header()
}

func (r ResponseWriter) Write(bytes []byte) (int, error) {
	return r.writer.Write(bytes)
}

func (r ResponseWriter) WriteHeader(statusCode int) {
	r.writer.WriteHeader(statusCode)
}

func (r ResponseWriter) StatusCode() int {
	return r.writer.StatusCode()
}
