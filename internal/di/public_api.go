package di

import (
	"io"
	"time"

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
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/internal/version"
	"github.com/spf13/afero"
)

func (c *Container) Args() []string {
	return c.args
}

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
	return c.cliOutput.GetOrBuild()
}

func (c *Container) RequestTracker() *server.RequestTracker {
	return c.requestTracker.GetOrBuild()
}

func (c *Container) HostCertManager() *server.HostCertManager {
	return c.hostCertManager.GetOrBuild()
}

func (c *Container) OptionsMiddleware(cfg config.OptionsHandling) contracts.Middleware {
	return infra.NewPrefixedMiddleware(
		options.NewMiddleware(
			options.WithHeaders(cfg.Headers),
			options.WithCode(cfg.Code),
		),
		"OPTIONS",
	)
}

func (c *Container) StaticMiddleware(path string, dir config.StaticDirectory) contracts.Middleware {
	return infra.NewPrefixedMiddleware(
		static.NewStaticMiddleware(
			static.WithFileSystem(afero.NewBasePathFs(c.fs, dir.Dir)),
			static.WithIndex(dir.Index),
			static.WithPrefix(path),
		),
		"STATIC",
	)
}

func (c *Container) VersionChecker(proxy string) *version.Checker {
	return version.NewVersionChecker(
		version.WithOutput(c.CliOutput()),
		version.WithHTTPClient(infra.MakeHTTPClient(proxy)),
		version.WithCurrentVersion(c.version),
	)
}

func (c *Container) MockHandler(response *config.Response) contracts.Handler {
	prefix := "MOCK"

	return infra.WithPrefix(prefix, mock.NewMockHandler(
		mock.WithResponse(response),
		mock.WithFileSystem(c.fs),
		mock.WithAfter(time.After),
	))
}

func (c *Container) ScriptHandler(scriptConfig *config.Script) contracts.Handler {
	prefix := "SCRIPT"
	output := c.CliOutput()

	return infra.WithPrefix(prefix, script.NewHandler(
		script.WithOutput(output.NewPrefixOutput(prefix)),
		script.WithScript(scriptConfig),
		script.WithFileSystem(c.fs),
	))
}

func (c *Container) RewriteMiddleware(rewriting *config.RewritingOption) contracts.Middleware {
	return infra.NewPrefixedMiddleware(
		rewrite.NewMiddleware(rewrite.WithRewritingOptions(rewriting)),
		"REWRITE",
	)
}

// ProxyHandler builds the fallthrough handler for a set of mappings. The HTTP
// client is passed in rather than created here, because its connection pool has
// a configuration lifetime and must be released with the generation that owns
// it.
func (c *Container) ProxyHandler(mappings config.Mappings, client contracts.HTTPClient) contracts.Handler {
	prefix := "PROXY"
	output := c.CliOutput()

	return infra.WithPrefix(prefix, proxy.NewProxyHandler(
		proxy.WithURLReplacerFactory(urlreplacer.NewURLReplacerFactory(mappings)),
		proxy.WithHTTPClient(client),
		proxy.WithOutput(output.NewPrefixOutput(prefix)),
	))
}
