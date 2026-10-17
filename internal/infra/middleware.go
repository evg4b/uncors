package infra

import (
	"github.com/evg4b/uncors/internal/contracts"
)

func Wrap(middleware contracts.Middleware, handler contracts.Handler) contracts.Handler {
	return HandlerFunc(func(w contracts.ResponseWriter, r *contracts.Request) error {
		return middleware.ServeHTTP(w, r, handler.ServeHTTP)
	})
}
