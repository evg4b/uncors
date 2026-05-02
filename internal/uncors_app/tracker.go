package uncorsapp

import (
	"net/url"
	"sync/atomic"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
)

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
		events: make(chan requestEvent, 50),
	}
}

func (t *requestTracker) Wrap(h contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(w contracts.ResponseWriter, r *contracts.Request) {
		id := t.nextID.Add(1)
		t.events <- requestEvent{
			id:        id,
			method:    r.Method,
			url:       r.URL,
			startedAt: time.Now(),
		}

		h.ServeHTTP(w, r)

		t.events <- requestEvent{id: id, done: true}
	})
}
