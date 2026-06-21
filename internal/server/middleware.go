package server

import "github.com/evg4b/uncors/internal/contracts"

func Mddleware(middlaware contracts.Middleware, handler contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(w contracts.ResponseWriter, r *contracts.Request) error {
		return middlaware.ServeHTTP(w, r, Next(handler))
	})
}

func Next(handler contracts.Handler) contracts.Next {
	return func(writer contracts.ResponseWriter, request *contracts.Request) error {
		return handler.ServeHTTP(writer, request)
	}
}
