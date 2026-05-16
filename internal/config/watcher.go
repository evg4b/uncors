package config

import (
	"fmt"
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
)

const debounceDelay = 10 * time.Millisecond

type Watcher struct {
	fsWatcher *fsnotify.Watcher
	onChange  func()
	done      chan struct{}
}

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
