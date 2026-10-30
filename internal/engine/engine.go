package engine

import (
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/evg4b/uncors/internal/certs"
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
	version   string
	certs     *certs.Manager
	server    *server.Server

	// Serialises Start, Restart, Shutdown and Close for their whole duration:
	// guarding only the generation swap lets overlapping restarts orphan listeners.
	mu         sync.Mutex
	generation *Generation
	tlsHosts   []string
	retiring   sync.WaitGroup
}

type Option = func(*Engine)

func WithPublisher(publisher events.Publisher) Option {
	return func(e *Engine) {
		e.publisher = publisher
	}
}

func WithVersion(version string) Option {
	return func(e *Engine) {
		e.version = version
	}
}

func New(fs afero.Fs, options ...Option) *Engine {
	engine := helpers.ApplyOptions(&Engine{
		fs:        fs,
		publisher: events.NoopPublisher{},
	}, options)

	engine.certs = certs.NewManager(engine.fs, engine.publisher)
	engine.server = server.New(engine.certs.GetCertificate, engine.publisher)

	return engine
}

func (e *Engine) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	start := func(ctx context.Context, targets []server.Target) (<-chan struct{}, error) {
		return nil, e.server.Start(ctx, targets)
	}

	return e.apply(ctx, "engine.start", uncorsConfig, start, events.ServerStarted{Mappings: uncorsConfig.Mappings})
}

func (e *Engine) Restart(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	served := events.ServerRestarted{Mappings: uncorsConfig.Mappings}

	return e.apply(ctx, "engine.restart", uncorsConfig, e.server.Restart, served)
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
	e.retiring.Wait()
}

// When serve fails the new generation is released and the current one is
// kept: server.Restart relies on it to serve the previous targets again.
func (e *Engine) apply(
	ctx context.Context,
	span string,
	uncorsConfig *config.UncorsConfig,
	serve func(context.Context, []server.Target) (<-chan struct{}, error),
	served events.Event,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	err := ctx.Err()
	if err != nil {
		return err
	}

	return telemetry.Run(ctx, span, func(ctx context.Context) error {
		generation, err := e.buildGeneration(ctx, uncorsConfig)
		if err != nil {
			return err
		}

		tlsHosts := httpsHosts(uncorsConfig.Mappings)
		e.certs.SetHosts(slices.Concat(e.tlsHosts, tlsHosts))

		retired, err := serve(ctx, generation.Targets())
		if err != nil {
			e.certs.SetHosts(e.tlsHosts)

			return errors.Join(err, generation.Discard())
		}

		previous := e.swap(generation)
		if previous != nil {
			previous.discardReplacedHAR(generation)
		}

		e.tlsHosts = tlsHosts
		e.certs.SetHosts(tlsHosts)
		e.publisher.Publish(served)

		if retired == nil {
			return closeGeneration(ctx, previous)
		}

		e.retire(ctx, previous, retired)

		return nil
	})
}

func (e *Engine) retire(ctx context.Context, previous *Generation, retired <-chan struct{}) {
	ctx = context.WithoutCancel(ctx)

	e.retiring.Go(func() {
		<-retired

		err := closeGeneration(ctx, previous)
		if err != nil {
			telemetry.Error(ctx, "failed to release the previous configuration", telemetry.Err(err))
			e.publisher.Publish(events.Notice{
				Level: events.NoticeError,
				Text:  "Failed to release the previous configuration: " + err.Error(),
			})
		}
	})
}

func (e *Engine) stop(ctx context.Context, span string, release func(context.Context) error) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return telemetry.Run(ctx, span, func(ctx context.Context) error {
		err := release(ctx)

		e.retiring.Wait()

		return errors.Join(err, closeGeneration(ctx, e.swap(nil)))
	})
}

func (e *Engine) swap(generation *Generation) *Generation {
	previous := e.generation
	e.generation = generation

	return previous
}

func httpsHosts(mappings config.Mappings) []string {
	hosts := make([]string, 0, len(mappings))

	for _, mapping := range mappings {
		if mapping.From.Scheme == "https" {
			hosts = append(hosts, mapping.From.Hostname)
		}
	}

	return hosts
}

func closeGeneration(ctx context.Context, generation *Generation) error {
	if generation == nil {
		return nil
	}

	return telemetry.Run(ctx, "generation.close", func(context.Context) error {
		return generation.Close()
	})
}
