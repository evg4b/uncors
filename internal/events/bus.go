package events

import (
	"sync"
	"sync/atomic"
)

// Bus fans events out to every subscriber. Publish never blocks: each
// subscriber has its own buffer, and an event that does not fit is dropped and
// counted for that subscriber only. Reporting is a presentation concern and
// must never slow down proxied traffic.
type Bus struct {
	mu          sync.RWMutex
	closed      bool
	subscribers map[*Subscription]struct{}
}

func NewBus() *Bus {
	return &Bus{subscribers: map[*Subscription]struct{}{}}
}

// Publish delivers event to every current subscriber without blocking.
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for subscription := range b.subscribers {
		subscription.deliver(event)
	}
}

// Subscribe registers a subscriber whose channel buffers up to buffer events.
// Events published before the call are not replayed. Subscribing to a closed
// bus returns a subscription whose channel is already closed.
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

// Close closes every subscription. Later events are discarded. It is safe to
// call more than once.
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

// Subscription is one subscriber's view of the bus.
type Subscription struct {
	bus     *Bus
	events  chan Event
	dropped atomic.Uint64
}

// Events returns the channel events arrive on. It is closed when the
// subscription or the bus is closed.
func (s *Subscription) Events() <-chan Event {
	return s.events
}

// Dropped returns how many events did not fit into the buffer.
func (s *Subscription) Dropped() uint64 {
	return s.dropped.Load()
}

// Close stops delivery and closes the events channel. It is safe to call more
// than once and after the bus is closed.
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
