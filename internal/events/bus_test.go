package events_test

import (
	"sync"
	"testing"

	"github.com/evg4b/uncors/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func drain(subscription *events.Subscription) []events.Event {
	var received []events.Event

	for {
		select {
		case event, ok := <-subscription.Events():
			if !ok {
				return received
			}

			received = append(received, event)
		default:
			return received
		}
	}
}

func TestBus(t *testing.T) {
	t.Run("delivers every event to every subscriber in order", func(t *testing.T) {
		bus := events.NewBus()
		first := bus.Subscribe(10)
		second := bus.Subscribe(10)

		bus.Publish(events.RequestStarted{ID: 1})
		bus.Publish(events.RequestCompleted{ID: 1})

		expected := []events.Event{events.RequestStarted{ID: 1}, events.RequestCompleted{ID: 1}}
		assert.Equal(t, expected, drain(first))
		assert.Equal(t, expected, drain(second))
	})

	t.Run("does not replay events published before subscribing", func(t *testing.T) {
		bus := events.NewBus()
		bus.Publish(events.RequestStarted{ID: 1})

		subscription := bus.Subscribe(10)

		assert.Empty(t, drain(subscription))
	})

	t.Run("drops and counts events a subscriber has no room for", func(t *testing.T) {
		bus := events.NewBus()
		slow := bus.Subscribe(1)
		fast := bus.Subscribe(10)

		for id := range uint64(3) {
			bus.Publish(events.RequestStarted{ID: id})
		}

		assert.Equal(t, []events.Event{events.RequestStarted{ID: 0}}, drain(slow))
		assert.Equal(t, uint64(2), slow.Dropped())
		assert.Len(t, drain(fast), 3)
		assert.Zero(t, fast.Dropped())
	})

	t.Run("closing a subscription stops delivery to it only", func(t *testing.T) {
		bus := events.NewBus()
		closed := bus.Subscribe(10)
		open := bus.Subscribe(10)

		closed.Close()
		closed.Close()
		bus.Publish(events.RequestStarted{ID: 1})

		_, ok := <-closed.Events()
		assert.False(t, ok)
		assert.Len(t, drain(open), 1)
	})

	t.Run("closing the bus closes every subscription", func(t *testing.T) {
		bus := events.NewBus()
		subscription := bus.Subscribe(10)

		bus.Close()
		bus.Close()
		bus.Publish(events.RequestStarted{ID: 1})
		subscription.Close()

		_, ok := <-subscription.Events()
		assert.False(t, ok)

		late := bus.Subscribe(10)
		_, ok = <-late.Events()
		assert.False(t, ok)
	})

	t.Run("is safe for concurrent publish, subscribe and close", func(_ *testing.T) {
		bus := events.NewBus()

		var waitGroup sync.WaitGroup

		for range 8 {
			waitGroup.Go(func() {
				subscription := bus.Subscribe(4)
				for id := range uint64(100) {
					bus.Publish(events.RequestStarted{ID: id})
				}

				subscription.Close()
			})
		}

		waitGroup.Wait()
		bus.Close()
	})
}

func TestHandlerKindString(t *testing.T) {
	require.Equal(t, "PROXY", events.HandlerProxy.String())
	require.Equal(t, "CACHE", events.HandlerCache.String())
	require.Equal(t, "OPTIONS", events.HandlerOptions.String())
	require.Equal(t, "NONE", events.HandlerNone.String())
	require.Equal(t, "UNKNOWN", events.HandlerKind(200).String())
}
