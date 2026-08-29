package di

import (
	"io"
	"net/http"
	"time"

	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/internal/handler/options"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/handler/script"
	"github.com/evg4b/uncors/internal/handler/static"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/internal/version"
	"github.com/spf13/afero"
)

func (c *Container) Fs() afero.Fs {
	return c.fs
}

func (c *Container) Version() string {
	return c.version
}

func (c *Container) Stdout() io.Writer {
	return c.stdout
}

func (c *Container) CliOutput() contracts.Output {
	return c.cliOutput()
}

func (c *Container) RequestTracker() *server.RequestTracker {
	return c.requestTracker()
}

func (c *Container) GenerateCertsCommand() *commands.GenerateCertsCommand {
	return c.generateCertsCommand()
}

func (c *Container) Server() *server.Server {
	return c.server()
}

func (c *Container) HostCertManager() *server.HostCertManager {
	return c.hostCertManager()
}

func (c *Container) OptionsMiddleware(cfg config.OptionsHandling) contracts.Middleware {
	return infra.NewPrefixedMiddleware(
		options.NewMiddleware(
			options.WithHeaders(cfg.Headers),
			options.WithCode(cfg.Code),
		).Wrap,
		styles.OptionsStyle.Render("OPTIONS"),
	)
}

func (c *Container) StaticMiddleware(path string, dir config.StaticDirectory) contracts.Middleware {
	return infra.NewPrefixedMiddleware(
		static.NewStaticMiddleware(
			static.WithFileSystem(afero.NewBasePathFs(c.fs, dir.Dir)),
			static.WithIndex(dir.Index),
			static.WithPrefix(path),
		).Wrap,
		styles.StaticStyle.Render("STATIC"),
	)
}

func (c *Container) VersionChecker(proxy string) *version.Checker {
	return version.NewVersionChecker(
		version.WithOutput(c.CliOutput()),
		version.WithHTTPClient(infra.MakeHTTPClient(proxy)),
		version.WithCurrentVersion(c.version),
	)
}

func (c *Container) MockHandler(response *config.Response) http.Handler {
	prefix := styles.MockStyle.Render("MOCK")

	return infra.WithPrefix(prefix, mock.NewMockHandler(
		mock.WithResponse(response),
		mock.WithFileSystem(c.fs),
		mock.WithAfter(time.After),
	))
}

func (c *Container) ScriptHandler(scriptConfig *config.Script) http.Handler {
	prefix := styles.RewriteStyle.Render("SCRIPT")
	output := c.CliOutput()

	return infra.WithPrefix(prefix, script.NewHandler(
		script.WithOutput(output.NewPrefixOutput(prefix)),
		script.WithScript(scriptConfig),
		script.WithFileSystem(c.fs),
	))
}

func (c *Container) RewriteMiddleware(rewriting *config.RewritingOption) contracts.Middleware {
	return infra.NewPrefixedMiddleware(
		rewrite.NewMiddleware(rewrite.WithRewritingOptions(rewriting)).Wrap,
		styles.RewriteStyle.Render("REWRITE"),
	)
}

func (c *Container) ProxyHandler(mappings config.Mappings, proxyURL string) http.Handler {
	prefix := styles.ProxyStyle.Render("PROXY")
	output := c.CliOutput()

	return infra.WithPrefix(prefix, proxy.NewProxyHandler(
		proxy.WithURLReplacerFactory(urlreplacer.NewURLReplacerFactory(mappings)),
		proxy.WithHTTPClient(infra.MakeHTTPClient(proxyURL)),
		proxy.WithOutput(output.NewPrefixOutput(prefix)),
	))
}
