package di

import (
	"github.com/evg4b/uncors/internal/contracts"
)

func (c *Container) Override(action ContainerOption) {
	action(c)
}

// WithCliOutput replaces the console output. It must be applied before
// anything resolves CliOutput: the container caches singletons on first use, so
// a late override would silently keep the old value and send the interactive
// mode's output to the terminal underneath the TUI. That failure is invisible
// at runtime, so it panics instead.
func WithCliOutput(factory func() contracts.Output) ContainerOption {
	return func(c *Container) {
		if c.cliOutput.Built() {
			panic("di: CliOutput was overridden after it had already been built")
		}

		c.cliOutput = newFactory(factory)
	}
}
