package di

import (
	"io"

	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
)

func (c *Container) Fs() afero.Fs {
	return c.fs
}

func (c *Container) Stdout() io.Writer {
	return c.stdout
}

func (c *Container) CliOutput() *tui.CliOutput {
	return c.cliOutput.GetOrBuild()
}

func (c *Container) RequestTracker() *server.RequestTracker {
	return c.requestTracker.GetOrBuild()
}

func (c *Container) GenerateCertsCommand() *commands.GenerateCertsCommand {
	return c.generateCertsCommand.GetOrBuild()
}

func (c *Container) HostCertManager() *server.HostCertManager {
	return c.hostCertManager.GetOrBuild()
}
