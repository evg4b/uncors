package script

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/spf13/afero"
)

type HandlerOption = func(*Handler)

func WithLogger(logger contracts.Logger) HandlerOption {
	return func(h *Handler) {
		h.logger = logger
	}
}

func WithScript(script config.Script) HandlerOption {
	return func(h *Handler) {
		h.script = script
	}
}

func WithFileSystem(fs afero.Fs) HandlerOption {
	return func(h *Handler) {
		h.fs = fs
	}
}
