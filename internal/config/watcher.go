package config

import (
	"fmt"
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
)

// debounceDelay is the wait time after the last file event before calling onChange.
// This prevents multiple rapid callbacks when editors write files in stages.
const debounceDelay = 10 * time.Millisecond

// ConfigWatcher watches a configuration file for changes and invokes a callback
// whenever the file is written or recreated. It uses a short debounce window to
// coalesce bursts of filesystem events that editors typically produce on save.
type ConfigWatcher struct {
	watcher  *fsnotify.Watcher
	onChange func()
	done     chan struct{}
}

// NewConfigWatcher creates a ConfigWatcher that monitors the given file path.
// onChange is called (after debouncing) on every write or create event.
// The returned watcher is already running; call Close to stop it.
func NewConfigWatcher(filePath string, onChange func()) (*ConfigWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	if err := w.Add(filePath); err != nil {
		_ = w.Close()

		return nil, fmt.Errorf("failed to watch config file '%s': %w", filePath, err)
	}

	cw := &ConfigWatcher{
		watcher:  w,
		onChange: onChange,
		done:     make(chan struct{}),
	}

	go cw.run()

	return cw, nil
}

// Close stops the watcher and releases all associated resources.
func (cw *ConfigWatcher) Close() error {
	close(cw.done)

	return cw.watcher.Close()
}

func (cw *ConfigWatcher) run() {
	var debounce *time.Timer

	stopDebounce := func() {
		if debounce != nil {
			debounce.Stop()
		}
	}

	for {
		select {
		case <-cw.done:
			stopDebounce()

			return

		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				stopDebounce()
				debounce = time.AfterFunc(debounceDelay, cw.onChange)
			}

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}

			log.Printf("config watcher error: %v", err)
		}
	}
}
