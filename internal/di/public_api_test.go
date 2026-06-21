package di_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/version"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestContainer(t *testing.T) {
	container := di.NewContainer()

	t.Run("fs", func(t *testing.T) {
		fs := container.Fs()

		assert.NotNil(t, fs)
		assert.IsType(t, &afero.MemMapFs{}, fs)
	})

	t.Run("stdout", func(t *testing.T) {
		stdout := container.Stdout()

		assert.NotNil(t, stdout)
	})

	t.Run("cli output", func(t *testing.T) {
		output := container.CliOutput()

		assert.NotNil(t, output)
		assert.Implements(t, (*contracts.Output)(nil), output)
	})

	t.Run("request tracker", func(t *testing.T) {
		tracker := container.RequestTracker()

		assert.NotNil(t, tracker)
		assert.IsType(t, &server.RequestTracker{}, tracker)
	})

	t.Run("generate certs command", func(t *testing.T) {
		cmd := container.GenerateCertsCommand()

		assert.NotNil(t, cmd)
		assert.IsType(t, &commands.GenerateCertsCommand{}, cmd)
	})

	t.Run("host cert manager", func(t *testing.T) {
		manager := container.HostCertManager()

		assert.NotNil(t, manager)
		assert.IsType(t, &server.HostCertManager{}, manager)
	})

	t.Run("options middleware", func(t *testing.T) {
		cfg := config.OptionsHandling{
			Headers: map[string]string{"X-Test": "value"},
			Code:    200,
		}
		middleware := container.OptionsMiddleware(cfg)

		assert.NotNil(t, middleware)
		assert.Implements(t, (*contracts.Middleware)(nil), middleware)
	})

	t.Run("static middleware", func(t *testing.T) {
		cfg := config.StaticDirectory{
			Dir:   "/public",
			Index: "index.html",
		}
		middleware := container.StaticMiddleware("/static", cfg)

		assert.NotNil(t, middleware)
		assert.Implements(t, (*contracts.Middleware)(nil), middleware)
	})

	t.Run("version checker", func(t *testing.T) {
		checker := container.VersionChecker("")

		assert.NotNil(t, checker)
		assert.IsType(t, &version.Checker{}, checker)
	})

	t.Run("singleton behavior", func(t *testing.T) {
		output1 := container.CliOutput()
		output2 := container.CliOutput()

		assert.Same(t, output1, output2)

		tracker1 := container.RequestTracker()
		tracker2 := container.RequestTracker()

		assert.Same(t, tracker1, tracker2)
	})
}
