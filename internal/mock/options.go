package mock

import "github.com/evg4b/uncors/internal/contracts"

type HandlerOption = func(*Handler)

func WithMock(mock Mock) HandlerOption {
	return func(handler *Handler) {
		handler.mock = mock
	}
}

func WithLogger(logger contracts.Logger) HandlerOption {
	return func(handler *Handler) {
		handler.logger = logger
	}
}
