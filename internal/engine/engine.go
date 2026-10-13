package engine

import (
	"context"
	"errors"
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

func WithFs(fs afero.Fs) Option {
	return func(e *Engine) {
		e.fs = fs
	}
}

func WithPublisher(publisher events.Publisher) Option {
	return func(e *Engine) {
		e.publisher = publisher
	}
}

func New(options ...Option) *Engine {
	engine := helpers.ApplyOptions(&Engine{
		fs:        afero.NewMemMapFs(),
		publisher: events.NoopPublisher{},
	}, options)

	engine.server = server.New(server.NewHostCertManager(engine.fs, engine.publisher), engine.publisher)

	return engine
}

func (e *Engine) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return telemetry.Run(ctx, "engine.start", func(ctx context.Context) error {
		generation := e.buildGeneration(ctx, uncorsConfig)

		err := e.server.Start(ctx, generation.Targets())
		if err != nil {
			return errors.Join(err, generation.Close())
		}

		e.swap(generation)
		e.publisher.Publish(events.ServerStarted{Mappings: uncorsConfig.Mappings})

		return nil
	})
}

func (e *Engine) Restart(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return telemetry.Run(ctx, "engine.restart", func(ctx context.Context) error {
		generation := e.buildGeneration(ctx, uncorsConfig)

		err := e.server.Restart(ctx, generation.Targets())
		if err != nil {
			return errors.Join(err, generation.Close())
		}

		previous := e.swap(generation)
		e.publisher.Publish(events.ServerRestarted{Mappings: uncorsConfig.Mappings})

		return closeGeneration(ctx, previous)
	})
}

func (e *Engine) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return telemetry.Run(ctx, "engine.shutdown", func(ctx context.Context) error {
		return errors.Join(
			e.server.Shutdown(ctx),
			closeGeneration(ctx, e.swap(nil)),
		)
	})
}

func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return telemetry.Run(context.Background(), "engine.close", func(ctx context.Context) error {
		return errors.Join(
			e.server.Close(),
			closeGeneration(ctx, e.swap(nil)),
		)
	})
}

func (e *Engine) Wait() {
	e.server.Wait()
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
