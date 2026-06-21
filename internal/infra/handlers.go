package infra

import (
	"errors"
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
)

var ErrResponseNotCasted = errors.New("received incorrect response writer type")

// MiddlewareFunc adapts an ordinary func into a Middleware.
type MiddlewareFunc func(contracts.Handler) contracts.Handler

func (f MiddlewareFunc) Wrap(next contracts.Handler) contracts.Handler {
	return f(next)
}

type HandlerFunc func(contracts.ResponseWriter, *contracts.Request) error

func (f HandlerFunc) ServeHTTP(w contracts.ResponseWriter, r *contracts.Request) error {
	return f(w, r)
}

func CastToHTTPHandler(handler contracts.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		writer, ok := response.(contracts.ResponseWriter)
		if !ok {
			panic(ErrResponseNotCasted)
		}

		err := handler.ServeHTTP(writer, request)
		if err != nil {
			HTTPError(writer, err)
		}
	})
}

func CastToContractsHandler(handler http.Handler) contracts.Handler {
	return HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) error {
		handler.ServeHTTP(writer, request)

		return nil
	})
}
