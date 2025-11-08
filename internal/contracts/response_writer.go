package contracts

import "net/http"

type ResponseWriter interface {
	http.ResponseWriter
	StatusCode() int
}

type ResponseWriterWrap struct {
	http.ResponseWriter

	Code int
}

func WrapResponseWriter(writer http.ResponseWriter) *ResponseWriterWrap {
	return &ResponseWriterWrap{ResponseWriter: writer}
}

func (r *ResponseWriterWrap) WriteHeader(statusCode int) {
	r.Code = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *ResponseWriterWrap) StatusCode() int {
	return r.Code
}
