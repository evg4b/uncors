package uncors

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
)

var errEmptyConfiguration = errors.New("configuration loader returned no configuration")

// ConfigLoader loads and validates the current configuration. It must report a
// malformed configuration as an error rather than as a nil result.
type ConfigLoader = func() (*config.UncorsConfig, error)

// Reloader owns configuration hot reload for every run mode. It watches the
// config file, loads and validates the new configuration and applies it to the
// running app.
//
// A configuration that fails to load is reported and discarded: the proxy keeps
// serving the previous one. Keeping this in one place is what stops the two run
// modes from growing two different reload contracts.
type Reloader struct {
	app        *Uncors
	output     contracts.Output
	load       ConfigLoader
	configPath string

	mu      sync.Mutex
	watcher *config.Watcher
}

func NewReloader(app *Uncors, output contracts.Output, load ConfigLoader, configPath string) *Reloader {
	return &Reloader{
		app:        app,
		output:     output,
		load:       load,
		configPath: configPath,
	}
}

// Start begins watching the config file. It is a no-op when the process runs
// without a config file.
func (r *Reloader) Start(ctx context.Context) error {
	if r.configPath == "" {
		return nil
	}

	watcher := config.NewWatcher(r.configPath)

	err := watcher.Watch(ctx, func() {
		reloadErr := r.Reload(ctx)
		if reloadErr != nil {
			r.report(reloadErr)
		}
	})
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.watcher = watcher

	return nil
}

// Reload loads the configuration and applies it to the running app. It is used
// both by the file watcher and by the interactive restart command.
func (r *Reloader) Reload(ctx context.Context) error {
	uncorsConfig, err := r.load()
	if err != nil {
		return err
	}

	if uncorsConfig == nil {
		return errEmptyConfiguration
	}

	return r.app.Restart(ctx, uncorsConfig)
}

// Watching reports whether the config file is currently being watched.
func (r *Reloader) Watching() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.watcher != nil
}

func (r *Reloader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.watcher == nil {
		return nil
	}

	watcher := r.watcher
	r.watcher = nil

	return watcher.Close()
}

func (r *Reloader) report(err error) {
	slog.Error("config reloading failed", "err", err)
	r.output.Errorf("Config reloading error: %v", err)
}
