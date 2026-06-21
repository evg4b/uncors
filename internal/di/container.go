package di

import (
	"io"

	"github.com/evg4b/uncors/internal/commands"
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
	}

	container = helpers.ApplyOptions(container, options)

	container.cliOutput = newFactory(container.newCliOutput)
	container.requestTracker = newFactory(server.NewRequestTracker)
	container.generateCertsCommand = newFactory(container.newGenerateCertsCommand)
	container.hostCertManager = newFactory(container.newHostCertManager)
	container.server = newFactory(container.newServer)

	return container
}
