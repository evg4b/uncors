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

type IRequestTracker interface {
	Events() <-chan RequestEvent
	Close()
	Emit(event RequestEvent)
}

type requestTracker struct {
	events chan RequestEvent
}

func NewRequestTracker() IRequestTracker {
	return &requestTracker{
		events: make(chan RequestEvent, requestEventsBufferSize),
	}
}

func (t *requestTracker) Events() <-chan RequestEvent {
	return t.events
}

func (t *requestTracker) Close() {
	close(t.events)
}

func (t *requestTracker) Emit(event RequestEvent) {
	t.events <- event
}
