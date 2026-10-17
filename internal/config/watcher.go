package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/fsnotify/fsnotify"
	"go.opentelemetry.io/otel/attribute"
)

const debounceDelay = 100 * time.Millisecond

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

func (w *Watcher) Watch(ctx context.Context, onChange func(), onError func(error)) error {
	if w.isWatching.Load() {
		return errAlreadyWatching
	}

	if w.filePath == "" {
		return nil
	}

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

	w.fsWatcher = fsWatcher
	w.isWatching.Store(true)

	go w.run(ctx, w.traced(ctx, onChange), onError)

	return nil
}

func (w *Watcher) Close() error {
	if w.fsWatcher != nil {
		return w.fsWatcher.Close()
	}

	return nil
}

func (w *Watcher) traced(ctx context.Context, onChange func()) func() {
	return func() {
		_, span := telemetry.Start(context.WithoutCancel(ctx), "config.watcher.event")
		defer span.End()

		span.SetAttributes(attribute.String("uncors.config.path", w.filePath))

		onChange()
	}
}

func (w *Watcher) run(ctx context.Context, onChange func(), onError func(error)) {
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

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			w.handleEvent(event, &debounce, onChange)

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}

			if onError != nil {
				onError(err)
			}
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
