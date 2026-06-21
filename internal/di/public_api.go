package di

import (
	"io"

	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/options"
	"github.com/evg4b/uncors/internal/handler/static"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/version"
	"github.com/spf13/afero"
)

func (c *Container) Fs() afero.Fs {
	return c.fs
}

func (c *Container) Stdout() io.Writer {
	return c.stdout
}

func (c *Container) CliOutput() contracts.Output {
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

func (c *Container) OptionsMiddleware(cfg config.OptionsHandling) contracts.Middleware {
	return options.NewMiddleware(
		options.WithHeaders(cfg.Headers),
		options.WithCode(cfg.Code),
	)
}

func (c *Container) StaticMiddleware(path string, dir config.StaticDirectory) contracts.Middleware {
	return static.NewStaticMiddleware(
		static.WithFileSystem(afero.NewBasePathFs(c.fs, dir.Dir)),
		static.WithIndex(dir.Index),
		static.WithPrefix(path),
	)
}

func (c *Container) VersionChecker(proxy string) *version.Checker {
	return version.NewVersionChecker(
		version.WithOutput(c.CliOutput()),
		version.WithHTTPClient(infra.MakeHTTPClient(proxy)),
		version.WithCurrentVersion(c.version),
	)
}
