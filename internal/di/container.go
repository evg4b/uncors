package di

import (
	"errors"
	"io"
	"slices"
	"sync"

	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/server"
	"github.com/spf13/afero"
)

type Container struct {
	fs      afero.Fs
	stdout  io.Writer
	args    []string
	version string

	cliOutput            factory[contracts.Output]
	requestTracker       factory[*server.RequestTracker]
	generateCertsCommand factory[*commands.GenerateCertsCommand]
	hostCertManager      factory[*server.HostCertManager]
	server               factory[*server.Server]
	proxy                factory[*Proxy]

	closersMu sync.Mutex
	closers   []io.Closer
}

type ContainerOption = func(c *Container)

func WithStdout(stdout io.Writer) ContainerOption {
	return func(c *Container) {
		c.stdout = stdout
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

func WithArgs(args []string) ContainerOption {
	return func(c *Container) {
		c.args = args
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

	container.cliOutput = newFactory(container.newCliOutput)
	container.requestTracker = newFactory(container.newRequestTracker)
	container.generateCertsCommand = newFactory(container.newGenerateCertsCommand)
	container.hostCertManager = newFactory(container.newHostCertManager)
	container.server = newFactory(container.newServer)
	container.proxy = newFactory(container.newProxy)

	return container
}

// Close releases every process-lifetime resource the container built, in
// reverse creation order so that a resource is never closed before the ones
// depending on it. It is safe to call more than once.
func (c *Container) Close() error {
	c.closersMu.Lock()
	closers := c.closers
	c.closers = nil
	c.closersMu.Unlock()

	errs := make([]error, 0, len(closers))

	for _, closer := range slices.Backward(closers) {
		err := closer.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// registerCloser binds a process-lifetime resource to the container, so that
// Close releases it. Factories are built lazily and potentially from different
// goroutines, so the list is guarded.
func (c *Container) registerCloser(closer io.Closer) {
	c.closersMu.Lock()
	defer c.closersMu.Unlock()

	c.closers = append(c.closers, closer)
}

// closerFunc adapts a plain shutdown function to io.Closer, for resources whose
// own Close reports nothing.
type closerFunc func() error

func (f closerFunc) Close() error { return f() }
