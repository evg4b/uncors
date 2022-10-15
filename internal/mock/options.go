package mock

type HandlerOption = func(*Handler)

func WithMock(mock Mock) HandlerOption {
	return func(handler *Handler) {
		handler.mock = mock
	}
}
