// Package engine runs the proxy without any user interface. It turns a
// configuration into listeners and handlers, swaps them on reload, and reports
// what happens only through the events it publishes.
package engine

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/logging"
	"github.com/evg4b/uncors/internal/server"
	"github.com/spf13/afero"
)

// Engine serves one configuration generation and owns the transition between
// generations. Releasing a generation is what flushes its HAR writers and frees
// its response cache, so exactly one generation must be alive per running set
// of targets.
//
// Start, Restart, Shutdown and Close may be called from independent goroutines,
// so mu serialises them for their whole duration. Guarding only the generation
// swap is not enough: two overlapping restarts would race on the server's
// listeners, orphaning one set of them, and could leave the active generation
// different from the one being served.
type Engine struct {
	fs        afero.Fs
	logger    *slog.Logger
	publisher events.Publisher
	server    *server.Server

	mu         sync.Mutex
	generation *Generation
}

type Option = func(*Engine)

// WithFs sets the file system that mocks, scripts, statics and the local CA
// are read from. The default is an empty in-memory file system.
func WithFs(fs afero.Fs) Option {
	return func(e *Engine) {
		e.fs = fs
	}
}

// WithLogger sets the diagnostic logger. The default discards every record.
func WithLogger(logger *slog.Logger) Option {
	return func(e *Engine) {
		e.logger = logger
	}
}

// WithPublisher sets where events go. The default discards them.
func WithPublisher(publisher events.Publisher) Option {
	return func(e *Engine) {
		e.publisher = publisher
	}
}

func New(options ...Option) *Engine {
	engine := helpers.ApplyOptions(&Engine{
		fs:        afero.NewMemMapFs(),
		logger:    logging.Discard(),
		publisher: events.NoopPublisher{},
	}, options)

	engine.server = server.New(server.NewHostCertManager(engine.fs), engine.publisher)

	return engine
}

// Start serves uncorsConfig and publishes ServerStarted.
func (e *Engine) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	generation, err := e.BuildGeneration(uncorsConfig)
	if err != nil {
		return err
	}

	err = e.server.Start(ctx, generation.Targets())
	if err != nil {
		return errors.Join(err, generation.Close())
	}

	e.swap(generation)
	e.publisher.Publish(events.ServerStarted{Mappings: uncorsConfig.Mappings})

	return nil
}

// Restart builds the new generation before touching the running one, so a
// config that fails to build leaves the engine serving the previous generation
// untouched. The old generation is released only once the new one is live.
func (e *Engine) Restart(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	generation, err := e.BuildGeneration(uncorsConfig)
	if err != nil {
		return err
	}

	err = e.server.Restart(ctx, generation.Targets())
	if err != nil {
		return errors.Join(err, generation.Close())
	}

	previous := e.swap(generation)
	e.publisher.Publish(events.ServerRestarted{Mappings: uncorsConfig.Mappings})

	return closeGeneration(previous)
}

// Shutdown stops accepting requests, waits for in-flight ones until ctx is
// done, and releases the active generation. It is safe to call more than once.
func (e *Engine) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return errors.Join(
		e.server.Shutdown(ctx),
		closeGeneration(e.swap(nil)),
	)
}

// Close stops the listeners immediately and releases the active generation.
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return errors.Join(
		e.server.Close(),
		closeGeneration(e.swap(nil)),
	)
}

// Wait blocks until every listener has stopped.
func (e *Engine) Wait() {
	e.server.Wait()
}

// swap installs generation as the active one and returns the one it replaced,
// which the caller is responsible for closing. Callers hold mu.
func (e *Engine) swap(generation *Generation) *Generation {
	previous := e.generation
	e.generation = generation

	return previous
}

func closeGeneration(generation *Generation) error {
	if generation == nil {
		return nil
	}

	return generation.Close()
}
