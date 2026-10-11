package cli

import (
	"io"
	"log/slog"

	"github.com/evg4b/uncors/internal/logging"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
)

// Env is what a command needs from the process it runs in.
type Env struct {
	Fs      afero.Fs
	Stdout  io.Writer
	Args    []string
	Version string
	// Logger receives diagnostics. Nil discards them.
	Logger *slog.Logger
}

func (e Env) usage(flags *pflag.FlagSet) {
	tui.PrintUsage(flags, e.Version)
}

func (e Env) logger() *slog.Logger {
	if e.Logger == nil {
		return logging.Discard()
	}

	return e.Logger
}
