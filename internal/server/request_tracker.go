package server

import (
	"net/url"
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

type RequestTracker struct {
	events chan RequestEvent
}

func NewRequestTracker() *RequestTracker {
	return &RequestTracker{
		events: make(chan RequestEvent, requestEventsBufferSize),
	}
}

func (t *RequestTracker) Events() <-chan RequestEvent {
	return t.events
}

func (t *RequestTracker) Close() {
	close(t.events)
}

func (t *RequestTracker) Emit(event RequestEvent) {
	select {
	case t.events <- event:
	default:
	}
}
