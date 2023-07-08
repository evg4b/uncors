package contracts

import (
	"errors"
	"net/http"
)

type Request = http.Request

type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}

type MiddlewareHandler interface {
	Wrap(next Handler) Handler
}

type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	f(w, r)
}

var ErrResponseNotCasted = errors.New("received incorrect response writer type")

func CastToHTTPHandler(handler Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		writer, ok := response.(ResponseWriter)
		if !ok {
			panic(ErrResponseNotCasted)
		}

		handler.ServeHTTP(writer, request)
	})
}
