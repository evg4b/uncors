package events_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errInvalidConfig = errors.New("invalid config")

func count(subscription *events.Subscription) int {
	received := 0
	for range subscription.Events() {
		received++
	}

	return received
}

func collect(bus *events.Bus, subscription *events.Subscription) []events.Event {
	bus.Close()

	var received []events.Event
	for event := range subscription.Events() {
		received = append(received, event)
	}

	return received
}

func receive(t *testing.T, subscription *events.Subscription) events.Event {
	t.Helper()

	select {
	case event := <-subscription.Events():
		return event
	case <-time.After(5 * time.Second):
		require.FailNow(t, "no event delivered")

		return nil
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
		assert.Equal(t, expected, collect(bus, first))
		assert.Equal(t, expected, collect(bus, second))
	})

	t.Run("does not replay events published before subscribing", func(t *testing.T) {
		bus := events.NewBus()
		bus.Publish(events.RequestStarted{ID: 1})

		subscription := bus.Subscribe(10)

		assert.Empty(t, collect(bus, subscription))
	})

	t.Run("drops and counts request events a subscriber has no room for", func(t *testing.T) {
		bus := events.NewBus()
		slow := bus.Subscribe(1)
		fast := bus.Subscribe(10)

		for id := range uint64(3) {
			bus.Publish(events.RequestStarted{ID: id})
		}

		assert.Equal(t, []events.Event{events.RequestStarted{ID: 0}}, collect(bus, slow))
		assert.Equal(t, uint64(2), slow.Dropped())
		assert.Len(t, collect(bus, fast), 3)
		assert.Zero(t, fast.Dropped())
	})
}

func TestBusOverflow(t *testing.T) {
	t.Run("delivers lifecycle events in order when the buffer is full", func(t *testing.T) {
		bus := events.NewBus()
		subscription := bus.Subscribe(1)

		bus.Publish(events.RequestStarted{ID: 1})
		bus.Publish(events.ServerStarted{})
		bus.Publish(events.RequestStarted{ID: 2})
		bus.Publish(events.ReloadFailed{Err: errInvalidConfig})
		bus.Publish(events.Notice{Level: events.NoticeError, Text: "certificate"})
		bus.Publish(events.NewVersionAvailable{Current: "1", Latest: "2"})
		bus.Publish(events.ServerRestarted{})
		bus.Publish(events.RequestStarted{ID: 3})
		bus.Publish(events.ServerStopped{})

		assert.Equal(t, []events.Event{
			events.RequestStarted{ID: 1},
			events.ServerStarted{},
			events.ReloadFailed{Err: errInvalidConfig},
			events.Notice{Level: events.NoticeError, Text: "certificate"},
			events.NewVersionAvailable{Current: "1", Latest: "2"},
			events.ServerRestarted{},
			events.ServerStopped{},
		}, collect(bus, subscription))
		assert.Equal(t, uint64(2), subscription.Dropped())
	})

	t.Run("does not block the publisher on lifecycle events", func(t *testing.T) {
		bus := events.NewBus()
		subscription := bus.Subscribe(1)

		for range 10_000 {
			bus.Publish(events.Notice{})
		}

		bus.Publish(events.ServerStopped{})

		received := collect(bus, subscription)
		require.Len(t, received, 10_001)
		assert.Equal(t, events.ServerStopped{}, received[len(received)-1])
		assert.Zero(t, subscription.Dropped())
	})

	t.Run("drops the completion of a request whose start was dropped", func(t *testing.T) {
		bus := events.NewBus()
		subscription := bus.Subscribe(1)

		bus.Publish(events.RequestStarted{ID: 1})
		bus.Publish(events.RequestStarted{ID: 2})
		bus.Publish(events.RequestCompleted{ID: 2})
		bus.Publish(events.RequestCompleted{ID: 1})

		assert.Equal(t, []events.Event{
			events.RequestStarted{ID: 1},
			events.RequestCompleted{ID: 1},
		}, collect(bus, subscription))
		assert.Equal(t, uint64(2), subscription.Dropped())
	})

	t.Run("delivers the completion of every delivered start", func(t *testing.T) {
		bus := events.NewBus()
		subscription := bus.Subscribe(3)

		for id := range uint64(10) {
			bus.Publish(events.RequestStarted{ID: id})
		}

		for id := range uint64(10) {
			bus.Publish(events.RequestCompleted{ID: id})
		}

		received := collect(bus, subscription)

		started := map[uint64]bool{}
		completed := map[uint64]bool{}

		for _, event := range received {
			switch typed := event.(type) {
			case events.RequestStarted:
				started[typed.ID] = true
			case events.RequestCompleted:
				completed[typed.ID] = true
			}
		}

		assert.Len(t, started, 3)
		assert.Equal(t, started, completed)
		assert.Equal(t, uint64(14), subscription.Dropped())
	})

	t.Run("drops a completion it never saw start", func(t *testing.T) {
		bus := events.NewBus()
		subscription := bus.Subscribe(10)

		bus.Publish(events.RequestCompleted{ID: 1})

		assert.Empty(t, collect(bus, subscription))
	})
}

func TestBusSubscription(t *testing.T) {
	t.Run("makes room as the subscriber reads", func(t *testing.T) {
		bus := events.NewBus()
		subscription := bus.Subscribe(2)

		bus.Publish(events.RequestStarted{ID: 1})
		bus.Publish(events.RequestCompleted{ID: 1})
		assert.Equal(t, events.RequestStarted{ID: 1}, receive(t, subscription))
		assert.Equal(t, events.RequestCompleted{ID: 1}, receive(t, subscription))

		bus.Publish(events.RequestStarted{ID: 2})

		received := collect(bus, subscription)
		require.NotEmpty(t, received)
		assert.Equal(t, events.RequestStarted{ID: 2}, received[0])
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
		assert.Len(t, collect(bus, open), 1)
	})

	t.Run("closing a subscription releases undelivered events", func(t *testing.T) {
		bus := events.NewBus()
		subscription := bus.Subscribe(10)

		bus.Publish(events.ServerStarted{})
		bus.Publish(events.ServerStopped{})
		subscription.Close()

		assert.LessOrEqual(t, count(subscription), 2)
		bus.Close()
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
		late.Close()
	})

	t.Run("is safe for concurrent publish, subscribe and close", func(_ *testing.T) {
		bus := events.NewBus()

		var waitGroup sync.WaitGroup

		for range 8 {
			waitGroup.Go(func() {
				subscription := bus.Subscribe(4)
				for id := range uint64(100) {
					bus.Publish(events.RequestStarted{ID: id})
					bus.Publish(events.Notice{})
				}

				subscription.Close()
			})
		}

		reader := bus.Subscribe(4)

		waitGroup.Go(func() {
			count(reader)
		})

		bus.Publish(events.ServerStopped{})
		bus.Close()
		waitGroup.Wait()
	})
}

func TestHandlerKindString(t *testing.T) {
	require.Equal(t, "PROXY", events.HandlerProxy.String())
	require.Equal(t, "CACHE", events.HandlerCache.String())
	require.Equal(t, "OPTNS", events.HandlerOptions.String())
	require.Equal(t, "NONE", events.HandlerNone.String())
	require.Equal(t, "UNKNOWN", events.HandlerKind(200).String())
}

func BenchmarkPublish(b *testing.B) {
	bus := events.NewBus()
	subscription := bus.Subscribe(1000)

	go count(subscription)

	b.ReportAllocs()

	var id uint64
	for b.Loop() {
		id++
		bus.Publish(events.RequestStarted{ID: id})
		bus.Publish(events.RequestCompleted{ID: id})
	}

	b.StopTimer()
	bus.Close()
}
