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

// Watcher monitors a configuration file for changes and invokes a callback
// whenever the file is written or recreated. It uses a short debounce window to
// coalesce bursts of filesystem events that editors typically produce on save.
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	onChange  func()
	done      chan struct{}
}

// NewWatcher creates a Watcher that monitors the given file path.
// onChange is called (after debouncing) on every write or create event.
// The returned watcher is already running; call Close to stop it.
func NewWatcher(filePath string, onChange func()) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	err = fsWatcher.Add(filePath)
	if err != nil {
		_ = fsWatcher.Close()

		return nil, fmt.Errorf("failed to watch config file '%s': %w", filePath, err)
	}

	watcher := &Watcher{
		fsWatcher: fsWatcher,
		onChange:  onChange,
		done:      make(chan struct{}),
	}

	go watcher.run()

	return watcher, nil
}

// Close stops the watcher and releases all associated resources.
func (cw *Watcher) Close() error {
	close(cw.done)

	return cw.fsWatcher.Close()
}

func (cw *Watcher) run() {
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

		case event, ok := <-cw.fsWatcher.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				stopDebounce()

				debounce = time.AfterFunc(debounceDelay, cw.onChange)
			}

		case err, ok := <-cw.fsWatcher.Errors:
			if !ok {
				return
			}

			log.Printf("config watcher error: %v", err)
		}
	}
}
