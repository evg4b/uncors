package server_test

import (
	"sync"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestTrackerEmit(t *testing.T) {
	t.Run("never blocks when nobody consumes the events", func(t *testing.T) {
		const eventsCount = 10_000

		tracker := server.NewRequestTracker()

		done := make(chan struct{})

		go func() {
			defer close(done)

			for index := range uint64(eventsCount) {
				tracker.Emit(server.RequestEvent{ID: index})
			}
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Emit blocked when the event buffer was full")
		}

		assert.Positive(t, tracker.Dropped(), "overflowing events must be counted as dropped")
	})

	t.Run("delivers events to the consumer", func(t *testing.T) {
		tracker := server.NewRequestTracker()

		tracker.Emit(server.RequestEvent{ID: 1})

		event := <-tracker.Events()

		assert.Equal(t, uint64(1), event.ID)
		assert.Zero(t, tracker.Dropped())
	})

	t.Run("is safe to close while producers are emitting", func(_ *testing.T) {
		tracker := server.NewRequestTracker()

		var waitGroup sync.WaitGroup

		waitGroup.Go(func() {
			for index := range uint64(1000) {
				tracker.Emit(server.RequestEvent{ID: index})
			}
		})

		tracker.Close()
		waitGroup.Wait()
	})

	t.Run("close is idempotent", func(t *testing.T) {
		tracker := server.NewRequestTracker()

		require.NotPanics(t, func() {
			tracker.Close()
			tracker.Close()
		})
	})
}

func TestNoopRequestSink(t *testing.T) {
	t.Run("discards events", func(t *testing.T) {
		require.NotPanics(t, func() {
			server.NoopRequestSink{}.Emit(server.RequestEvent{ID: 1})
		})
	})
}
