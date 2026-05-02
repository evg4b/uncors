package uncorsapp

import (
	"net/url"
	"sync/atomic"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
)

const requestEventsBufferSize = 1000

type requestEvent struct {
	id        uint64
	method    string
	url       *url.URL
	startedAt time.Time
	done      bool
}

type requestTracker struct {
	events chan requestEvent
	nextID atomic.Uint64
}

func newRequestTracker() *requestTracker {
	return &requestTracker{
		events: make(chan requestEvent, requestEventsBufferSize),
	}
}

func (t *requestTracker) Wrap(handler contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) {
		requestID := t.nextID.Add(1)
		select {
		case t.events <- requestEvent{
			id:        requestID,
			method:    request.Method,
			url:       request.URL,
			startedAt: time.Now(),
		}:
		default:
		}

		handler.ServeHTTP(writer, request)

		select {
		case t.events <- requestEvent{id: requestID, done: true}:
		default:
		}
	})
}
