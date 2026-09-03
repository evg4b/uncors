// Package app owns the uncors application runtime: the active configuration,
// the reload lifecycle, the config watcher and the proxy generations derived
// from them.
//
// It is the single implementation both run modes drive. Interactive mode wraps
// it in a Bubble Tea client; non-interactive mode calls Run and renders the
// output stream directly. Nothing in this package may depend on a terminal.
package app

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
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

	watcher     *config.Watcher
	shutdownOne sync.Once
}

// New creates the service for one process. cfg is the configuration the first
// generation is built from; configPath is the file to watch, empty when no
// config file is in use.
func New(container *di.Container, cfg *config.UncorsConfig, configPath string, load Loader) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		container:  container,
		proxy:      container.Proxy(),
		configPath: configPath,
		load:       load,
		ctx:        ctx,
		cancel:     cancel,
		cfg:        cfg,
	}
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
	err := s.proxy.Start(ctx, s.Config())
	if err != nil {
		return err
	}

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
	})

	return err
}

// Close releases what the service itself owns. The generation and the server
// belong to the container, which closes them in turn.
func (s *Service) Close() error {
	s.cancel()

	if s.watcher != nil {
		return s.watcher.Close()
	}

	return nil
}

func (s *Service) reloadOnce() {
	output := s.container.CliOutput()

	reloaded, err := s.load()
	if err != nil {
		output.Errorf("Failed to reload config: %v", err)

		return
	}

	err = s.proxy.Restart(s.ctx, reloaded)
	if err != nil {
		output.Errorf("Failed to restart server: %v", err)

		return
	}

	s.mu.Lock()
	s.cfg = reloaded
	s.mu.Unlock()
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
		s.container.CliOutput().Errorf("Failed to watch config file: %v", err)

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

	select {
	case sig := <-stop:
		if sig == syscall.SIGINT {
			// Move past the "^C" the terminal echoed.
			_, _ = s.container.CliOutput().Write([]byte("\n"))
		}

		log.Println("shutdown signal received")
	case <-ctx.Done():
	case <-s.ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	_ = s.Shutdown(shutdownCtx)
}
