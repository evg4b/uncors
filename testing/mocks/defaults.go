package mocks

import (
	"io"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/tui"
)

func NoopOutput() contracts.Output {
	return tui.NewCliOutput(io.Discard)
}
