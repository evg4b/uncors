package di

import (
	"context"
	"errors"
	"sync"

	"github.com/evg4b/uncors/internal/config"
)

// Proxy serves one configuration generation on the container's server and owns
// the transition between generations. It reports nothing to the console:
// describing what is happening belongs to whoever is presenting the service. Releasing a generation is what flushes
// its HAR writers and frees its response cache, so exactly one generation must
// be alive per running set of targets.
//
// Start, Restart and Shutdown are driven by independent goroutines - the config
// watcher, the signal handler, the TUI - so mu guards the active generation.
type Proxy struct {
	container *Container

	mu      sync.Mutex
	runtime *Runtime
}

func (c *Container) newProxy() *Proxy {
	return &Proxy{container: c}
}

func (p *Proxy) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	runtime, err := p.container.BuildRuntime(uncorsConfig)
	if err != nil {
		return err
	}

	err = p.container.Server().Start(ctx, runtime.Targets())
	if err != nil {
		return errors.Join(err, runtime.Close())
	}

	p.swap(runtime)

	return nil
}

// Restart builds the new generation before touching the running one, so a
// config that fails to build leaves the proxy serving the previous generation
// untouched. The old generation is released only once the new one is live.
func (p *Proxy) Restart(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	runtime, err := p.container.BuildRuntime(uncorsConfig)
	if err != nil {
		return err
	}

	err = p.container.Server().Restart(ctx, runtime.Targets())
	if err != nil {
		return errors.Join(err, runtime.Close())
	}

	previous := p.swap(runtime)

	return closeRuntime(previous)
}

func (p *Proxy) Shutdown(ctx context.Context) error {
	return errors.Join(
		p.container.Server().Shutdown(ctx),
		closeRuntime(p.swap(nil)),
	)
}

func (p *Proxy) Close() error {
	return errors.Join(
		p.container.Server().Close(),
		closeRuntime(p.swap(nil)),
	)
}

func (p *Proxy) Wait() {
	p.container.Server().Wait()
}

// swap installs runtime as the active generation and returns the one it
// replaced, which the caller is responsible for closing.
func (p *Proxy) swap(runtime *Runtime) *Runtime {
	p.mu.Lock()
	defer p.mu.Unlock()

	previous := p.runtime
	p.runtime = runtime

	return previous
}

func closeRuntime(runtime *Runtime) error {
	if runtime == nil {
		return nil
	}

	return runtime.Close()
}
