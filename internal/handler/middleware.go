package handler

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infra"
)

func Mddleware(middlaware contracts.Middleware, handler contracts.Handler) contracts.Handler {
	return infra.HandlerFunc(func(w contracts.ResponseWriter, r *contracts.Request) error {
		return middlaware.ServeHTTP(w, r, handler.ServeHTTP)
	})
}
