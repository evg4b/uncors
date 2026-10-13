package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/spf13/afero"
)

type Engine struct {
	fs        afero.Fs
	publisher events.Publisher
	server    *server.Server

	// Serialises Start, Restart, Shutdown and Close for their whole duration:
	// guarding only the generation swap lets overlapping restarts orphan listeners.
	mu         sync.Mutex
	generation *Generation
}

type Option = func(*Engine)

func WithPublisher(publisher events.Publisher) Option {
	return func(e *Engine) {
		e.publisher = publisher
	}
}

// New builds an engine that reads static files, mocks, scripts, and the CA
// from fs.
func New(fs afero.Fs, options ...Option) *Engine {
	engine := helpers.ApplyOptions(&Engine{
		fs:        fs,
		publisher: events.NoopPublisher{},
	}, options)

	engine.server = server.New(server.NewHostCertManager(engine.fs, engine.publisher), engine.publisher)

	return engine
}

func (e *Engine) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	return e.apply(ctx, "engine.start", uncorsConfig, e.server.Start, func(mappings fmt.Stringer) events.Event {
		return events.ServerStarted{Mappings: mappings}
	})
}

func (e *Engine) Restart(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	return e.apply(ctx, "engine.restart", uncorsConfig, e.server.Restart, func(mappings fmt.Stringer) events.Event {
		return events.ServerRestarted{Mappings: mappings}
	})
}

func (e *Engine) Shutdown(ctx context.Context) error {
	return e.stop(ctx, "engine.shutdown", e.server.Shutdown)
}

func (e *Engine) Close() error {
	return e.stop(context.Background(), "engine.close", func(context.Context) error {
		return e.server.Close()
	})
}

func (e *Engine) Wait() {
	e.server.Wait()
}

// apply serves a generation built from uncorsConfig in place of the current
// one. When serve fails the new generation is released and the current one
// is kept.
func (e *Engine) apply(
	ctx context.Context,
	span string,
	uncorsConfig *config.UncorsConfig,
	serve func(context.Context, []server.Target) error,
	served func(fmt.Stringer) events.Event,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return telemetry.Run(ctx, span, func(ctx context.Context) error {
		generation := e.buildGeneration(ctx, uncorsConfig)

		err := serve(ctx, generation.Targets())
		if err != nil {
			return errors.Join(err, generation.Close())
		}

		previous := e.swap(generation)
		e.publisher.Publish(served(uncorsConfig.Mappings))

		return closeGeneration(ctx, previous)
	})
}

// stop releases the server with release, then the current generation.
func (e *Engine) stop(ctx context.Context, span string, release func(context.Context) error) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return telemetry.Run(ctx, span, func(ctx context.Context) error {
		return errors.Join(
			release(ctx),
			closeGeneration(ctx, e.swap(nil)),
		)
	})
}

func (e *Engine) swap(generation *Generation) *Generation {
	previous := e.generation
	e.generation = generation

	return previous
}

func closeGeneration(ctx context.Context, generation *Generation) error {
	if generation == nil {
		return nil
	}

	return telemetry.Run(ctx, "generation.close", func(context.Context) error {
		return generation.Close()
	})
}
