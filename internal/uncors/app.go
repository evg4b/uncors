package uncors

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"

	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/afero"
)

type Uncors struct {
	fs afero.Fs

	output    contracts.Output
	server    *server.Server
	container *di.Container

	// runtimeMu guards the currently active configuration generation, which is
	// replaced on every reload and released on shutdown.
	runtimeMu sync.Mutex
	runtime   *di.Runtime
}

func CreateUncors(container *di.Container) *Uncors {
	return &Uncors{
		fs:        container.Fs(),
		output:    container.CliOutput(),
		container: container,
		server:    container.Server(),
	}
}

func (app *Uncors) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	tui.PrintLogo(app.output, app.container.Version())
	app.output.Print("")
	app.output.WarnBox(tui.DisclaimerMessage)
	app.output.Print("")
	app.output.InfoBox(uncorsConfig.Mappings.String())
	app.output.Print("")

	app.warnAboutExposure(uncorsConfig.Listen)

	runtime, err := app.container.BuildRuntime(uncorsConfig)
	if err != nil {
		return err
	}

	err = app.server.Start(ctx, runtime.Targets())
	if err != nil {
		return errors.Join(err, runtime.Close())
	}

	app.swapRuntime(runtime)

	return nil
}

// Restart builds the new configuration generation before touching the running
// one, so a config that fails to build leaves the proxy serving the previous
// generation untouched. The old generation is released only once the new one is
// live, which is what flushes its HAR writers and frees its cache.
func (app *Uncors) Restart(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	app.output.Info("Restarting server....")

	runtime, err := app.container.BuildRuntime(uncorsConfig)
	if err != nil {
		return err
	}

	err = app.server.Restart(ctx, runtime.Targets())
	if err != nil {
		return errors.Join(err, runtime.Close())
	}

	previous := app.swapRuntime(runtime)

	app.output.InfoBox(
		"Server restarted",
		uncorsConfig.Mappings.String(),
	)

	return closeRuntime(previous)
}

func (app *Uncors) Close() error {
	return errors.Join(
		app.server.Close(),
		closeRuntime(app.swapRuntime(nil)),
	)
}

func (app *Uncors) Wait() {
	app.server.Wait()
}

func (app *Uncors) Shutdown(ctx context.Context) error {
	return errors.Join(
		app.server.Shutdown(ctx),
		closeRuntime(app.swapRuntime(nil)),
	)
}

// warnAboutExposure tells the user when the proxy is reachable beyond this
// machine. uncors strips CORS protections and can hold a trusted local CA, so
// binding anything but a loopback address deserves to be said out loud.
func (app *Uncors) warnAboutExposure(listen string) {
	address := net.ParseIP(listen)
	if address != nil && address.IsLoopback() {
		return
	}

	app.output.WarnBox(
		fmt.Sprintf("uncors is listening on %s and disables CORS protections.", listen),
		"Anyone who can reach this machine can use it as a proxy.",
		"Do not run it this way on an untrusted network.",
	)
	app.output.Print("")
}

// swapRuntime installs runtime as the active generation and returns the one it
// replaced, which the caller is responsible for closing.
func (app *Uncors) swapRuntime(runtime *di.Runtime) *di.Runtime {
	app.runtimeMu.Lock()
	defer app.runtimeMu.Unlock()

	previous := app.runtime
	app.runtime = runtime

	return previous
}

func closeRuntime(runtime *di.Runtime) error {
	if runtime == nil {
		return nil
	}

	return runtime.Close()
}
