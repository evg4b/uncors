package static

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/spf13/afero"
)

type HandlerOption = func(*Handler)

func WithFileSystem(fs afero.Fs) HandlerOption {
	return func(h *Handler) {
		h.fs = fs
	}
}

func WithIndex(index string) HandlerOption {
	return func(h *Handler) {
		h.index = index
	}
}

func WithNext(next contracts.Handler) HandlerOption {
	return func(h *Handler) {
		h.next = next
	}
}

func WithLogger(logger contracts.Logger) HandlerOption {
	return func(h *Handler) {
		h.logger = logger
	}
}

func WithPrefix(prefix string) HandlerOption {
	return func(h *Handler) {
		h.prefix = prefix
	}
}
