package server

import (
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
)

const requestEventsBufferSize = 1000

type RequestEvent struct {
	ID        uint64
	Method    string
	URL       *url.URL
	StartedAt time.Time
	Prefix    string
	Done      bool
	Data      *contracts.RequestData
}

// RequestSink receives request lifecycle events from the server. It is the only
// part of the tracker the request path depends on, and implementations must
// never block: activity reporting is a presentation concern and must not be able
// to apply backpressure to proxied traffic.
type RequestSink interface {
	Emit(event RequestEvent)
}

type IRequestTracker interface {
	RequestSink

	Events() <-chan RequestEvent
	Close()
}

// RequestTracker is a channel backed RequestSink. Events are delivered to a
// single consumer (the TUI or RequestPrinter); when the consumer cannot keep up
// events are dropped and counted rather than delaying the request that produced
// them.
type RequestTracker struct {
	mu      sync.RWMutex
	closed  bool
	events  chan RequestEvent
	dropped atomic.Uint64
}

func NewRequestTracker() *RequestTracker {
	return &RequestTracker{
		events: make(chan RequestEvent, requestEventsBufferSize),
	}
}

func (t *RequestTracker) Events() <-chan RequestEvent {
	return t.events
}

// Close stops event delivery and closes the events channel. It is safe to call
// concurrently with Emit and safe to call more than once.
func (t *RequestTracker) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return
	}

	t.closed = true
	close(t.events)
}

// Emit delivers the event to the consumer, dropping it when the buffer is full
// or the tracker is closed. It never blocks.
func (t *RequestTracker) Emit(event RequestEvent) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		t.dropped.Add(1)

		return
	}

	select {
	case t.events <- event:
	default:
		t.dropped.Add(1)
	}
}

// Dropped returns the number of events that were discarded because no consumer
// kept up with them.
func (t *RequestTracker) Dropped() uint64 {
	return t.dropped.Load()
}

// NoopRequestSink discards every event. It is the correct sink for tests and for
// any run mode that does not display request activity.
type NoopRequestSink struct{}

func (NoopRequestSink) Emit(RequestEvent) {}
