package cli

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/spf13/afero"
)

type GenerateCertsOption = func(*GenerateCertsCommand)

func WithOutput(output contracts.Output) GenerateCertsOption {
	return func(c *GenerateCertsCommand) {
		c.output = output
	}
}

func WithFs(fs afero.Fs) GenerateCertsOption {
	return func(c *GenerateCertsCommand) {
		c.fs = fs
	}
}
