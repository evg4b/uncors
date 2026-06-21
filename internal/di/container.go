package di

import (
	"io"

	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
)

type Container struct {
	fs     afero.Fs
	stdout io.Writer

	cliOutput            factory[tui.CliOutput]
	requestTracker       factory[server.RequestTracker]
	generateCertsCommand factory[commands.GenerateCertsCommand]
	hostCertManager      factory[server.HostCertManager]
	server               factory[server.Server]
}

func NewContainer(fs afero.Fs, stdout io.Writer) *Container {
	container := &Container{fs: fs, stdout: stdout}

	container.cliOutput = newFactory(container.newCliOutput)
	container.requestTracker = newFactory(server.NewRequestTracker)
	container.generateCertsCommand = newFactory(container.newGenerateCertsCommand)
	container.hostCertManager = newFactory(container.newHostCertManager)
	container.server = newFactory(container.newServer)

	return container
}
