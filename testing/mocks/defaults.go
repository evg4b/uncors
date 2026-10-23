package mocks

import (
	"io"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/tui/render"
)

func NoopOutput() contracts.Output {
	return render.NewCliOutput(io.Discard)
}
