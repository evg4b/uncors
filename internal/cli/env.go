package cli

import (
	"io"

	"github.com/evg4b/uncors/internal/tui/render"
	"github.com/evg4b/uncors/internal/version"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
)

type Env struct {
	Fs     afero.Fs
	Stdout io.Writer
	Args   []string
	Build  version.Build
}

func (e Env) usage(flags *pflag.FlagSet) {
	render.PrintUsage(flags, e.Build.Version())
}
