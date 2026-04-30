package commands

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/spf13/afero"
)

type Option = func(*GenerateCertsCommand)

func WithOutput(output contracts.Output) Option {
	return func(c *GenerateCertsCommand) {
		c.output = output
	}
}

func WithFs(fs afero.Fs) Option {
	return func(c *GenerateCertsCommand) {
		c.fs = fs
	}
}
