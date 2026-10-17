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
		bus:      b,
		events:   make(chan Event),
		wake:     make(chan struct{}, 1),
		done:     make(chan struct{}),
		limit:    buffer,
		admitted: map[uint64]struct{}{},
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		close(subscription.events)

		return subscription
	}

	b.subscribers[subscription] = struct{}{}

	go subscription.pump()

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
		subscription.finish()
	}

	clear(b.subscribers)
}

func (b *Bus) unsubscribe(subscription *Subscription) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.subscribers, subscription)
}

type Subscription struct {
	bus     *Bus
	events  chan Event
	wake    chan struct{}
	done    chan struct{}
	stop    sync.Once
	dropped atomic.Uint64

	mu       sync.Mutex
	queue    []Event
	head     int
	requests int
	limit    int
	admitted map[uint64]struct{}
	finished bool
}

func (s *Subscription) Events() <-chan Event {
	return s.events
}

func (s *Subscription) Dropped() uint64 {
	return s.dropped.Load()
}

func (s *Subscription) Close() {
	s.bus.unsubscribe(s)
	s.stop.Do(func() {
		close(s.done)
	})
}

func (s *Subscription) deliver(event Event) {
	s.mu.Lock()

	if !s.accept(event) {
		s.mu.Unlock()
		s.dropped.Add(1)

		return
	}

	s.push(event)
	s.mu.Unlock()
	s.signal()
}

func (s *Subscription) accept(event Event) bool {
	switch typed := event.(type) {
	case RequestStarted:
		if s.requests >= s.limit {
			return false
		}

		s.admitted[typed.ID] = struct{}{}
		s.requests++

		return true
	case RequestCompleted:
		if _, ok := s.admitted[typed.ID]; !ok {
			return false
		}

		delete(s.admitted, typed.ID)
		s.requests++

		return true
	default:
		return true
	}
}

func (s *Subscription) push(event Event) {
	if s.head > 0 && len(s.queue) == cap(s.queue) {
		kept := copy(s.queue, s.queue[s.head:])
		clear(s.queue[kept:])
		s.queue = s.queue[:kept]
		s.head = 0
	}

	s.queue = append(s.queue, event)
}

func (s *Subscription) finish() {
	s.mu.Lock()
	s.finished = true
	s.mu.Unlock()
	s.signal()
}

func (s *Subscription) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Subscription) pump() {
	defer close(s.events)

	for {
		event, ok := s.next()
		if !ok {
			return
		}

		select {
		case s.events <- event:
			s.delivered()
		case <-s.done:
			return
		}
	}
}

func (s *Subscription) next() (Event, bool) {
	for {
		s.mu.Lock()

		if s.head < len(s.queue) {
			event := s.queue[s.head]
			s.mu.Unlock()

			return event, true
		}

		finished := s.finished
		s.mu.Unlock()

		if finished {
			return nil, false
		}

		select {
		case <-s.wake:
		case <-s.done:
			return nil, false
		}
	}
}

func (s *Subscription) delivered() {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.queue[s.head].(type) {
	case RequestStarted, RequestCompleted:
		s.requests--
	}

	s.queue[s.head] = nil
	s.head++

	if s.head == len(s.queue) {
		s.queue = s.queue[:0]
		s.head = 0
	}
}
