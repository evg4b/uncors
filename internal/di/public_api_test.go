package di_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/version"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerOptions(t *testing.T) {
	t.Run("WithStdout", func(t *testing.T) {
		buf := &bytes.Buffer{}

		container := di.NewContainer(di.WithStdout(buf))
		defer testutils.Close(t, container)

		assert.Same(t, buf, container.Stdout())
	})

	t.Run("WithVersion", func(t *testing.T) {
		container := di.NewContainer(di.WithVersion("1.2.3"))
		defer testutils.Close(t, container)

		assert.Equal(t, "1.2.3", container.Version())
	})

	t.Run("WithFs", func(t *testing.T) {
		fs := afero.NewOsFs()

		container := di.NewContainer(di.WithFs(fs))
		defer testutils.Close(t, container)

		assert.Same(t, fs, container.Fs())
	})
}

func TestContainer(t *testing.T) {
	container := di.NewContainer()
	defer testutils.Close(t, container)

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

	t.Run("server", func(t *testing.T) {
		srv := container.Server()

		assert.NotNil(t, srv)
		assert.IsType(t, &server.Server{}, srv)
	})

	t.Run("options middleware", func(t *testing.T) {
		cfg := config.OptionsHandling{
			Headers: map[string]string{"X-Test": "value"},
			Code:    200,
		}
		middleware := container.OptionsMiddleware(cfg)

		assert.NotNil(t, middleware)
	})

	t.Run("static middleware", func(t *testing.T) {
		cfg := config.StaticDirectory{
			Dir:   "/public",
			Index: "index.html",
		}
		middleware := container.StaticMiddleware("/static", cfg)

		assert.NotNil(t, middleware)
	})

	t.Run("version checker", func(t *testing.T) {
		checker, err := container.VersionChecker("")

		require.NoError(t, err)
		assert.NotNil(t, checker)
		assert.IsType(t, &version.Checker{}, checker)
	})

	t.Run("mock handler", func(t *testing.T) {
		response := &config.Response{Code: 200, Raw: "ok"}
		handler := container.MockHandler(response)

		assert.NotNil(t, handler)
		assert.Implements(t, (*http.Handler)(nil), handler)
	})

	t.Run("script handler", func(t *testing.T) {
		script := &config.Script{
			Matcher: config.RequestMatcher{Path: "/api"},
			Script:  `response:set_status(200)`,
		}
		handler := container.ScriptHandler(script)

		assert.NotNil(t, handler)
		assert.Implements(t, (*http.Handler)(nil), handler)
	})

	t.Run("rewrite middleware", func(t *testing.T) {
		rewriting := &config.RewritingOption{From: "/old", To: "/new"}
		middleware := container.RewriteMiddleware(rewriting)

		assert.NotNil(t, middleware)
	})

	t.Run("proxy handler", func(t *testing.T) {
		mappings := config.Mappings{
			{From: hosts.Localhost.HTTP(), To: hosts.Localhost.HTTPS()},
		}
		handler, err := container.ProxyHandler(mappings, "")

		require.NoError(t, err)
		assert.NotNil(t, handler)
		assert.Implements(t, (*http.Handler)(nil), handler)
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

func TestContainerOutput(t *testing.T) {
	t.Run("uses the explicitly provided output", func(t *testing.T) {
		expected := tui.NewCliOutput(io.Discard)

		container := di.NewContainer(di.WithOutput(expected))
		defer testutils.Close(t, container)

		assert.Same(t, expected, container.CliOutput())
	})

	t.Run("falls back to a console output over stdout", func(t *testing.T) {
		buf := &bytes.Buffer{}

		container := di.NewContainer(di.WithStdout(buf))
		defer testutils.Close(t, container)

		container.CliOutput().Print("hello")

		assert.Contains(t, buf.String(), "hello")
	})
}

func TestContainerClose(t *testing.T) {
	t.Run("close with no closers succeeds", func(t *testing.T) {
		container := di.NewContainer()
		err := container.Close()

		require.NoError(t, err)
	})
}
