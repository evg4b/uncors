package config

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

const debounceDelay = 10 * time.Millisecond

var errAlreadyWatching = errors.New("watcher is already watching")

type Watcher struct {
	filePath   string
	isWatching atomic.Bool

	mu        sync.Mutex
	fsWatcher *fsnotify.Watcher
}

func NewWatcher(filePath string) *Watcher {
	return &Watcher{
		filePath: filePath,
	}
}

// Watch starts delivering debounced change notifications for the configured
// file. It is a no-op when no config file is in use.
func (w *Watcher) Watch(ctx context.Context, onChange func()) error {
	if w.filePath == "" {
		return nil
	}

	// Claim the watcher atomically: two concurrent Watch calls must not both
	// get past this point and leak an fsnotify watcher between them.
	if !w.isWatching.CompareAndSwap(false, true) {
		return errAlreadyWatching
	}

	err := w.start(ctx, onChange)
	if err != nil {
		w.isWatching.Store(false)

		return err
	}

	return nil
}

// Close stops the watcher and releases the claim taken by Watch, so a closed
// Watcher reports its true state rather than staying permanently "watching".
func (w *Watcher) Close() error {
	w.isWatching.Store(false)

	w.mu.Lock()
	fsWatcher := w.fsWatcher
	w.fsWatcher = nil
	w.mu.Unlock()

	if fsWatcher != nil {
		return fsWatcher.Close()
	}

	return nil
}

func (w *Watcher) start(ctx context.Context, onChange func()) error {
	_, err := os.Stat(w.filePath)
	if err != nil {
		return fmt.Errorf("failed to watch config file '%s': %w", w.filePath, err)
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Watch the parent directory rather than the file itself. Many editors save
	// via write-to-temp + rename, which replaces the file's inode; a watch bound
	// to the inode goes silent after the first save. Watching the directory and
	// filtering by file name survives atomic replaces.
	dir := filepath.Dir(w.filePath)

	err = fsWatcher.Add(dir)
	if err != nil {
		return errors.Join(
			fsWatcher.Close(),
			fmt.Errorf("failed to watch config directory '%s': %w", dir, err),
		)
	}

	w.mu.Lock()
	w.fsWatcher = fsWatcher
	w.mu.Unlock()

	// run owns the watcher it was handed rather than reading the field, so a
	// Close followed by a fresh Watch cannot race the previous run goroutine.
	go w.run(ctx, fsWatcher, onChange)

	return nil
}

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
