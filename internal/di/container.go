package di

import (
	"errors"
	"io"

	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/server"
	"github.com/spf13/afero"
)

type Container struct {
	fs      afero.Fs
	stdout  io.Writer
	version string

	cliOutput            factory[contracts.Output]
	requestTracker       factory[*server.RequestTracker]
	generateCertsCommand factory[*commands.GenerateCertsCommand]
	hostCertManager      factory[*server.HostCertManager]
	server               factory[*server.Server]
	cache                factory1[contracts.Cache, *config.CacheConfig]

	closers []io.Closer
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

func NewContainer(options ...ContainerOption) *Container {
	container := &Container{
		fs:      afero.NewMemMapFs(),
		stdout:  io.Discard,
		version: "0.0.0",
		closers: []io.Closer{},
	}

	container = helpers.ApplyOptions(container, options)

	container.cliOutput = newFactory(container.newCliOutput)
	container.requestTracker = newFactory(server.NewRequestTracker)
	container.generateCertsCommand = newFactory(container.newGenerateCertsCommand)
	container.hostCertManager = newFactory(container.newHostCertManager)
	container.server = newFactory(container.newServer)
	container.cache = newFactory1(container.newCache)

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
