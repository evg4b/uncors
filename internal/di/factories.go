package di

import (
	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/cache"
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

func (c *Container) newServer() *server.Server {
	return server.New(c.HostCertManager(), c.RequestTracker())
}

func (c *Container) newCache(cfs *config.CacheConfig) contracts.Cache {
	instance := cache.NewRistrettoCache(cfs.MaxSize, cfs.ExpirationTime)
	c.closers = append(c.closers, instance)

	return instance
}
