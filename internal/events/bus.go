package events

import (
	"sync"
	"sync/atomic"
)

type Bus struct {
	mu          sync.RWMutex
	closed      bool
	subscribers map[*Subscription]struct{}
}

func NewBus() *Bus {
	return &Bus{subscribers: map[*Subscription]struct{}{}}
}

func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for subscription := range b.subscribers {
		subscription.deliver(event)
	}
}

func (b *Bus) Subscribe(buffer int) *Subscription {
	subscription := &Subscription{
		bus:    b,
		events: make(chan Event, buffer),
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		close(subscription.events)

		return subscription
	}

	b.subscribers[subscription] = struct{}{}

	return subscription
}

func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.closed = true

	for subscription := range b.subscribers {
		close(subscription.events)
	}

	clear(b.subscribers)
}

func (b *Bus) unsubscribe(subscription *Subscription) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.subscribers[subscription]; !ok {
		return
	}

	delete(b.subscribers, subscription)
	close(subscription.events)
}

type Subscription struct {
	bus     *Bus
	events  chan Event
	dropped atomic.Uint64
}

func (s *Subscription) Events() <-chan Event {
	return s.events
}

func (s *Subscription) Dropped() uint64 {
	return s.dropped.Load()
}

func (s *Subscription) Close() {
	s.bus.unsubscribe(s)
}

// deliver runs under the bus read lock, so the channel cannot be closed
// concurrently.
func (s *Subscription) deliver(event Event) {
	select {
	case s.events <- event:
	default:
		s.dropped.Add(1)
	}
}
