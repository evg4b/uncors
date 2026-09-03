// Package app owns the uncors application runtime: the active configuration,
// the reload lifecycle, the config watcher and the proxy generations derived
// from them.
//
// It is the single implementation both run modes drive. Interactive mode wraps
// it in a Bubble Tea client; non-interactive mode calls Run and renders the
// output stream directly. Nothing in this package may depend on a terminal.
package app

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/server"
)

const (
	shutdownTimeout   = 15 * time.Second
	versionCheckDelay = 50 * time.Millisecond
)

// Loader produces the configuration for a new generation. It reports failure
// rather than returning a nil config, because a config that fails to parse or
// validate must leave the running generation untouched.
type Loader func() (*config.UncorsConfig, error)

// Service owns the application runtime. Its methods are safe to call from
// independent goroutines - the config watcher, a signal handler and a UI client
// all drive it concurrently.
type Service struct {
	container  *di.Container
	proxy      *di.Proxy
	configPath string
	load       Loader

	// The service outlives every individual call, so its lifetime context is
	// state rather than a parameter; clients derive their own from Context().
	ctx    context.Context //nolint:containedctx // service lifetime, not request scope
	cancel context.CancelFunc

	mu  sync.RWMutex
	cfg *config.UncorsConfig

	// reloadMu guards the coalescing state below, not the reload itself.
	reloadMu  sync.Mutex
	reloading bool
	pending   bool

	events  *emitter
	tracker server.IRequestTracker

	// inFlightMu guards the authoritative set of requests currently being
	// served. It lives here rather than in a UI widget because it is a fact
	// about the server, and because a reload has to be able to clear it.
	inFlightMu sync.RWMutex
	inFlight   map[uint64]server.RequestEvent

	watcher     *config.Watcher
	shutdownOne sync.Once
}

// New creates the service for one process. cfg is the configuration the first
// generation is built from; configPath is the file to watch, empty when no
// config file is in use.
func New(container *di.Container, cfg *config.UncorsConfig, configPath string, load Loader) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	service := &Service{
		container:  container,
		proxy:      container.Proxy(),
		configPath: configPath,
		load:       load,
		ctx:        ctx,
		cancel:     cancel,
		cfg:        cfg,
		events:     newEmitter(),
		tracker:    container.RequestTracker(),
		inFlight:   map[uint64]server.RequestEvent{},
	}

	// Start pumping before anything can serve a request, so no activity is
	// missed between construction and Start.
	go service.pumpRequests()

	return service
}

// InFlight returns the requests currently being served, oldest first. A client
// that connects late, or one that lost track, can rebuild its view from this
// rather than from the events it happened to witness.
func (s *Service) InFlight() []server.RequestEvent {
	s.inFlightMu.RLock()
	defer s.inFlightMu.RUnlock()

	requests := make([]server.RequestEvent, 0, len(s.inFlight))
	for _, request := range s.inFlight {
		requests = append(requests, request)
	}

	slices.SortFunc(requests, func(a, b server.RequestEvent) int {
		return cmp.Compare(a.ID, b.ID)
	})

	return requests
}

// Events returns the service's event stream. It has a single consumer: the TUI
// in interactive mode, the console renderer otherwise.
func (s *Service) Events() <-chan Event {
	return s.events.Events()
}

// Status returns the latest lifecycle state. It is always current, even when
// the notification carrying it was dropped.
func (s *Service) Status() Status {
	return s.events.Status()
}

// DroppedEvents reports how many events were discarded because the presenter
// could not keep up.
func (s *Service) DroppedEvents() uint64 {
	return s.events.Dropped()
}

// Context returns the service lifetime context. It is cancelled when the
// service shuts down, so clients can use it to stop their own work.
func (s *Service) Context() context.Context {
	return s.ctx
}

// Config returns the configuration of the current generation.
func (s *Service) Config() *config.UncorsConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg
}

// Start brings up the first generation and, once it is serving, begins watching
// the config file and checking for a newer release. It returns when the
// listeners are bound.
func (s *Service) Start(ctx context.Context) error {
	cfg := s.Config()

	s.events.EmitLifecycle(LifecycleEvent{State: StateStarting, Mappings: cfg.Mappings})

	err := s.proxy.Start(ctx, cfg)
	if err != nil {
		s.events.EmitLifecycle(LifecycleEvent{State: StateStartFailed, Err: err})

		return err
	}

	s.events.EmitLifecycle(LifecycleEvent{State: StateStarted, Mappings: cfg.Mappings})

	s.startWatching()

	go s.checkVersion()

	return nil
}

// Reload rebuilds the runtime from the current config file.
//
// Reloads are serialised, and requests arriving while one is running are
// coalesced into a single follow-up run rather than queueing one run each. A
// failure to load, validate or bind leaves the running generation serving.
func (s *Service) Reload() {
	s.reloadMu.Lock()

	if s.reloading {
		// Someone else is mid-reload; hand them the work and return.
		s.pending = true
		s.reloadMu.Unlock()

		return
	}

	s.reloading = true
	s.reloadMu.Unlock()

	for {
		s.reloadOnce()

		s.reloadMu.Lock()

		if !s.pending {
			// Clearing the flag under the same lock that sets it is what stops a
			// request arriving here from being dropped.
			s.reloading = false
			s.reloadMu.Unlock()

			return
		}

		s.pending = false
		s.reloadMu.Unlock()
	}
}

// Wait blocks until every listener has stopped.
func (s *Service) Wait() {
	s.proxy.Wait()
}

// Run starts the service and blocks until it stops, either on a shutdown signal
// or when ctx is cancelled. It is the non-interactive entry point.
func (s *Service) Run(ctx context.Context) error {
	err := s.Start(ctx)
	if err != nil {
		return err
	}

	go s.awaitSignal(ctx)

	s.Wait()

	return nil
}

// Shutdown stops the server and releases the active generation. It is safe to
// call more than once and from more than one goroutine.
func (s *Service) Shutdown(ctx context.Context) error {
	var err error

	s.shutdownOne.Do(func() {
		s.cancel()

		err = s.proxy.Shutdown(ctx)

		s.events.EmitLifecycle(LifecycleEvent{State: StateStopped, Err: err})
	})

	return err
}

// Close releases what the service itself owns. The generation and the server
// belong to the container, which closes them in turn.
func (s *Service) Close() error {
	s.cancel()
	s.events.Close()

	if s.watcher != nil {
		return s.watcher.Close()
	}

	return nil
}

// pumpRequests owns the request tracker. It maintains the in-flight set and
// forwards every event to the presenter.
func (s *Service) pumpRequests() {
	for event := range s.tracker.Events() {
		s.inFlightMu.Lock()

		if event.Done {
			delete(s.inFlight, event.ID)
		} else if event.URL != nil {
			s.inFlight[event.ID] = event
		}

		s.inFlightMu.Unlock()

		s.events.send(RequestEvent{Event: event})
	}
}

func (s *Service) clearInFlight() {
	s.inFlightMu.Lock()
	defer s.inFlightMu.Unlock()

	clear(s.inFlight)
}

func (s *Service) reloadOnce() {
	reloaded, err := s.load()
	if err != nil {
		s.events.EmitLifecycle(LifecycleEvent{State: StateReloadFailed, Err: err})

		return
	}

	// Announced only once the config is known to be good, so a rejected config
	// never claims the server is restarting.
	s.events.EmitLifecycle(LifecycleEvent{State: StateReloading})

	err = s.proxy.Restart(s.ctx, reloaded)
	if err != nil {
		s.events.EmitLifecycle(LifecycleEvent{State: StateReloadFailed, Err: err})

		return
	}

	s.mu.Lock()
	s.cfg = reloaded
	s.mu.Unlock()

	// The generation those requests belonged to is gone; anything still tracked
	// against it would linger forever. This is why a reload triggered by a file
	// save now clears the view the same way the restart key always did.
	s.clearInFlight()

	s.events.EmitLifecycle(LifecycleEvent{State: StateReloaded, Mappings: reloaded.Mappings})
}

// startWatching begins reloading on config file changes. A missing or
// unwatchable file is reported and then ignored: the server keeps serving the
// configuration it already has.
func (s *Service) startWatching() {
	if s.configPath == "" {
		return
	}

	watcher := config.NewWatcher(s.configPath)

	err := watcher.Watch(s.ctx, s.Reload)
	if err != nil {
		s.events.EmitLog(LevelError, fmt.Sprintf("Failed to watch config file: %v", err))

		return
	}

	s.watcher = watcher
}

func (s *Service) checkVersion() {
	select {
	case <-time.After(versionCheckDelay):
	case <-s.ctx.Done():
		return
	}

	s.container.
		VersionChecker(s.Config().Proxy).
		CheckNewVersion(s.ctx)
}

// awaitSignal stops the service on the first OS signal or context
// cancellation. The shutdown itself runs on a fresh context, because ctx may be
// the one that triggered it.
func (s *Service) awaitSignal(ctx context.Context) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	defer signal.Stop(stop)

	interrupted := false

	select {
	case sig := <-stop:
		interrupted = sig == syscall.SIGINT

		log.Println("shutdown signal received")
	case <-ctx.Done():
	case <-s.ctx.Done():
	}

	// Interrupted tells the presenter the terminal already echoed "^C"; whether
	// that needs moving past is its decision, not the service's.
	s.events.EmitLifecycle(LifecycleEvent{State: StateStopping, Interrupted: interrupted})

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	_ = s.Shutdown(shutdownCtx)
}
