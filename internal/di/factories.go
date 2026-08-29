package di

import (
	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/server"
)

func (c *Container) newGenerateCertsCommand() *commands.GenerateCertsCommand {
	return commands.NewGenerateCertsCommand(
		commands.WithOutput(c.CliOutput()),
		commands.WithFs(c.fs),
	)
}

func (c *Container) newHostCertManager() *server.HostCertManager {
	return server.NewHostCertManager(c.fs)
}

func (c *Container) newServer() *server.Server {
	return server.New(c.HostCertManager(), c.RequestTracker())
}
