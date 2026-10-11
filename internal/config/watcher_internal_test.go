package config

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/require"
)

var errSyntheticWatcher = fmt.Errorf("synthetic test error")

func newIsolatedWatcher(t *testing.T) (*Watcher, chan fsnotify.Event, chan error) {
	t.Helper()

	fsW, err := fsnotify.NewWatcher()
	require.NoError(t, err)

	// Replace the channels the Watcher struct exposes. The kqueue backend holds
	// its own references to the original channels and will never touch these.
	eventsCh := make(chan fsnotify.Event)
	errsCh := make(chan error)
	fsW.Events = eventsCh
	fsW.Errors = errsCh

	watcher := &Watcher{
		filePath:  "test",
		fsWatcher: fsW,
	}
	watcher.isWatching.Store(true)

	return watcher, eventsCh, errsCh
}

func runAndWait(ctx context.Context, watcher *Watcher, onChange func()) <-chan struct{} {
	exited := make(chan struct{})

	go func() {
		defer close(exited)

		watcher.run(ctx, onChange)
	}()

	return exited
}

func awaitExit(t *testing.T, watcher *Watcher, exited <-chan struct{}) {
	t.Helper()

	select {
	case <-exited:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("run goroutine did not exit")
	}

	require.NoError(t, watcher.fsWatcher.Close())
}

func TestWatcherRunStopsWhenChannelCloses(t *testing.T) {
	t.Run("events", func(t *testing.T) {
		watcher, events, _ := newIsolatedWatcher(t)
		exited := runAndWait(t.Context(), watcher, func() {})

		close(events)

		awaitExit(t, watcher, exited)
	})

	t.Run("errors", func(t *testing.T) {
		watcher, _, errs := newIsolatedWatcher(t)
		exited := runAndWait(t.Context(), watcher, func() {})

		close(errs)

		awaitExit(t, watcher, exited)
	})
}

func TestWatcherRunErrorPath(t *testing.T) {
	watcher, _, errs := newIsolatedWatcher(t)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	exited := runAndWait(ctx, watcher, func() {})

	errs <- errSyntheticWatcher

	cancel()

	awaitExit(t, watcher, exited)
}
