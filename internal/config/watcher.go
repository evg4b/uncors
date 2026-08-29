package config

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

const debounceDelay = 10 * time.Millisecond

var errAlreadyWatching = errors.New("watcher is already watching")

type Watcher struct {
	filePath   string
	fsWatcher  *fsnotify.Watcher
	isWatching atomic.Bool
}

func NewWatcher(filePath string) *Watcher {
	return &Watcher{
		filePath: filePath,
	}
}

func (w *Watcher) Watch(ctx context.Context, onChange func()) error {
	if !w.isWatching.CompareAndSwap(false, true) {
		return errAlreadyWatching
	}

	_, err := os.Stat(w.filePath)
	if err != nil {
		w.isWatching.Store(false)

		return fmt.Errorf("failed to watch config file '%s': %w", w.filePath, err)
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		w.isWatching.Store(false)

		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Watch the parent directory rather than the file itself. Many editors save
	// via write-to-temp + rename, which replaces the file's inode; a watch bound
	// to the inode goes silent after the first save. Watching the directory and
	// filtering by file name survives atomic replaces.
	dir := filepath.Dir(w.filePath)

	err = fsWatcher.Add(dir)
	if err != nil {
		_ = fsWatcher.Close()

		w.isWatching.Store(false)

		return fmt.Errorf("failed to watch config directory '%s': %w", dir, err)
	}

	w.fsWatcher = fsWatcher

	go w.run(ctx, fsWatcher, onChange)

	return nil
}

// Close releases the file system watch. It is safe to call on a watcher that
// was never started, and a closed watcher can be started again.
func (w *Watcher) Close() error {
	if !w.isWatching.CompareAndSwap(true, false) {
		return nil
	}

	fsWatcher := w.fsWatcher
	w.fsWatcher = nil

	if fsWatcher == nil {
		return nil
	}

	return fsWatcher.Close()
}

// run owns the fsnotify watcher it was handed: reading it from the struct would
// race with Close clearing the field.
func (w *Watcher) run(ctx context.Context, fsWatcher *fsnotify.Watcher, onChange func()) {
	var debounce *time.Timer

	defer func() {
		if debounce != nil {
			debounce.Stop()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-fsWatcher.Events:
			if !ok {
				return
			}

			w.handleEvent(event, &debounce, onChange)

		case err, ok := <-fsWatcher.Errors:
			if !ok {
				return
			}

			log.Printf("config watcher error: %v", err)
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event, debounce **time.Timer, onChange func()) {
	if filepath.Base(event.Name) != filepath.Base(w.filePath) {
		return
	}

	if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
		if *debounce != nil {
			(*debounce).Stop()
		}

		*debounce = time.AfterFunc(debounceDelay, onChange)
	}
}
