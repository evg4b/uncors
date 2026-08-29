package di

import (
	"errors"
	"io"
	"sync"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
)

// Container is the composition root. It holds the process scoped values and the
// application scoped singletons; everything derived from a configuration lives
// in a Runtime instead, so that it can be released when that configuration stops
// being current.
type Container struct {
	fs      afero.Fs
	stdout  io.Writer
	output  contracts.Output
	version string
	caDir   string

	cliOutput       func() contracts.Output
	clients         func() *infra.ClientPool
	requestTracker  func() *server.RequestTracker
	hostCertManager func() *server.HostCertManager
	server          func() *server.Server

	closers []io.Closer
}

type ContainerOption = func(c *Container)

// WithStdout sets the writer the default console output writes to.
func WithStdout(stdout io.Writer) ContainerOption {
	return func(c *Container) {
		c.stdout = stdout
	}
}

// WithOutput sets the output implementation explicitly. Run modes that render
// their own output (the TUI) choose it here, before anything can capture the
// default one.
func WithOutput(output contracts.Output) ContainerOption {
	return func(c *Container) {
		c.output = output
	}
}

// WithCADir sets the directory holding the local CA. Empty means the default
// location, which follows $XDG_CONFIG_HOME and then the home directory.
func WithCADir(caDir string) ContainerOption {
	return func(c *Container) {
		c.caDir = caDir
	}
}

func WithVersion(version string) ContainerOption {
	return func(c *Container) {
		c.version = version
	}
}

func WithFs(fs afero.Fs) ContainerOption {
	return func(c *Container) {
		c.fs = fs
	}
}

func NewContainer(options ...ContainerOption) *Container {
	container := &Container{
		fs:      afero.NewMemMapFs(),
		stdout:  io.Discard,
		version: "0.0.0",
		closers: []io.Closer{},
	}

	container = helpers.ApplyOptions(container, options)

	// The client pool is process scoped: a transport owns a connection pool and
	// is meant to outlive any single configuration.
	container.closers = append(container.closers, closerFn(func() error {
		container.clients().CloseIdleConnections()

		return nil
	}))

	container.cliOutput = sync.OnceValue(container.newCliOutput)
	container.clients = sync.OnceValue(infra.NewClientPool)
	container.requestTracker = sync.OnceValue(server.NewRequestTracker)
	container.hostCertManager = sync.OnceValue(container.newHostCertManager)
	container.server = sync.OnceValue(container.newServer)

	return container
}

func (c *Container) Close() error {
	var errs []error

	for _, closer := range c.closers {
		err := closer.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// closerFn adapts a function to io.Closer.
type closerFn func() error

func (f closerFn) Close() error { return f() }

func (c *Container) newCliOutput() contracts.Output {
	if c.output != nil {
		return c.output
	}

	return tui.NewCliOutput(c.stdout)
}
