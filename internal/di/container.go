package di

import (
	"io"

	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
)

type Container struct {
	fs     afero.Fs
	stdout io.Writer

	cliOutput            factory[tui.CliOutput]
	requestTracker       factory[server.RequestTracker]
	generateCertsCommand factory[commands.GenerateCertsCommand]
}

func NewContainer(fs afero.Fs, stdout io.Writer) *Container {
	c := &Container{fs: fs, stdout: stdout}

	c.cliOutput = newFactory(c.newCliOutput)
	c.requestTracker = newFactory(server.NewRequestTracker)
	c.generateCertsCommand = newFactory(c.newGenerateCertsCommand)

	return c
}

func (c *Container) CliOutput() *tui.CliOutput {
	return c.cliOutput.GetOrBuild()
}

func (c *Container) newCliOutput() *tui.CliOutput {
	return tui.NewCliOutput(c.stdout)
}

func (c *Container) RequestTracker() *server.RequestTracker {
	return c.requestTracker.GetOrBuild()
}

func (c *Container) GenerateCertsCommand() *commands.GenerateCertsCommand {
	return c.generateCertsCommand.GetOrBuild()
}

func (c *Container) newGenerateCertsCommand() *commands.GenerateCertsCommand {
	return commands.NewGenerateCertsCommand(
		commands.WithOutput(c.CliOutput()),
		commands.WithFs(c.fs),
	)
}
