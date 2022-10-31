package mock

import "github.com/evg4b/uncors/internal/contracts"

type HandlerOption = func(*Handler)

func WithResponse(response Response) HandlerOption {
	return func(handler *Handler) {
		handler.response = response
	}
}

func WithLogger(logger contracts.Logger) HandlerOption {
	return func(handler *Handler) {
		handler.logger = logger
	}
}
