package di

import (
	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"
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

func (c *Container) Server() *server.Server {
	return c.server.GetOrBuild()
}

func (c *Container) newCliOutput() contracts.Output {
	return tui.NewCliOutput(c.stdout)
}

func (c *Container) Proxy() *Proxy {
	return c.proxy.GetOrBuild()
}

func (c *Container) newServer() *server.Server {
	// RequestTracker is resolved first and therefore registered first, so the
	// reverse-order Close stops the server before the sink it emits into.
	instance := server.New(c.HostCertManager(), c.RequestTracker())
	c.registerCloser(instance)

	return instance
}

func (c *Container) newRequestTracker() *server.RequestTracker {
	tracker := server.NewRequestTracker()

	c.registerCloser(closerFunc(func() error {
		tracker.Close()

		return nil
	}))

	return tracker
}
