package cli

import (
	"io"

	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
)

type Env struct {
	Fs      afero.Fs
	Stdout  io.Writer
	Args    []string
	Version string
}

func (e Env) usage(flags *pflag.FlagSet) {
	tui.PrintUsage(flags, e.Version)
}
