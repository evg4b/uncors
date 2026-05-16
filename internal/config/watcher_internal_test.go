package config

import (
	"fmt"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/require"
)

var errSyntheticWatcher = fmt.Errorf("synthetic test error")

// newIsolatedWatcher creates a Watcher whose run goroutine uses custom channels
// that are owned entirely by the test. Both fsnotify.Watcher.Events and
// fsnotify.Watcher.Errors are replaced with channels we control so that the
// fsnotify kqueue backend goroutine (which holds its own copies of the original
// channels) never writes to our replacements. This avoids any data race between
// the test and the kqueue backend.
//
// The underlying fsnotify watcher must be closed by the caller after run exits
// to release the backend goroutine cleanly.
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
		fsWatcher: fsW,
		onChange:  func() {},
		done:      make(chan struct{}),
	}

	return watcher, eventsCh, errsCh
}

// runAndWait starts watcher.run in a goroutine and returns a channel that is
// closed when run returns.
func runAndWait(watcher *Watcher) <-chan struct{} {
	exited := make(chan struct{})

	go func() {
		defer close(exited)

		watcher.run()
	}()

	return exited
}

// TestWatcherRunEventsNotOk covers the early return in run() when the Events
// channel is closed with ok=false (lines 75-77 in watcher.go).
func TestWatcherRunEventsNotOk(t *testing.T) {
	watcher, events, _ := newIsolatedWatcher(t)

	exited := runAndWait(watcher)

	// Closing the channel makes the Events select case fire with ok=false,
	// which triggers the return on lines 75-77.
	close(events)

	select {
	case <-exited:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("run goroutine did not exit after Events channel was closed")
	}

	// Close the underlying fsnotify watcher so its backend goroutine can exit.
	// It closes its own (original) Events and Errors channels, not ours.
	_ = watcher.fsWatcher.Close()
}

// TestWatcherRunErrorsNotOk covers the return in run() when the Errors channel
// is closed with ok=false (lines 86-88 in watcher.go).
func TestWatcherRunErrorsNotOk(t *testing.T) {
	watcher, _, errs := newIsolatedWatcher(t)

	exited := runAndWait(watcher)

	// Closing the channel makes the Errors select case fire with ok=false,
	// which triggers the return on lines 86-88.
	close(errs)

	select {
	case <-exited:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("run goroutine did not exit after Errors channel was closed")
	}

	// Close the underlying fsnotify watcher so its backend goroutine can exit.
	// It closes its own (original) Events and Errors channels, not ours.
	_ = watcher.fsWatcher.Close()
}

// TestWatcherRunErrorPath covers the log.Printf branch in run() when an error
// arrives from the backend with ok=true (line 90 in watcher.go).
func TestWatcherRunErrorPath(t *testing.T) {
	watcher, _, errs := newIsolatedWatcher(t)

	exited := runAndWait(watcher)

	// Sending to the unbuffered errs channel blocks until run's select receives
	// it. Because the channel is open, ok=true and the error is logged (line 90).
	errs <- errSyntheticWatcher

	// Signal run to stop and wait for it to exit cleanly.
	close(watcher.done)

	select {
	case <-exited:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("run goroutine did not exit after done was closed")
	}

	// Close the underlying fsnotify watcher so its backend goroutine can exit.
	_ = watcher.fsWatcher.Close()
}
