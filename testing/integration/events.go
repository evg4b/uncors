//go:build integration

package integration

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/stretchr/testify/require"
)

const (
	eventsBuffer  = 10_000
	eventWait     = 2 * time.Second
	eventPollTick = 5 * time.Millisecond
)

type Request struct {
	Started   events.RequestStarted
	Completed events.RequestCompleted
}

type eventRecorder struct {
	mu       sync.Mutex
	received []events.Event
	done     chan struct{}
}

func recordEvents(subscription *events.Subscription) *eventRecorder {
	recorder := &eventRecorder{done: make(chan struct{})}

	go func() {
		defer close(recorder.done)

		for event := range subscription.Events() {
			recorder.mu.Lock()
			recorder.received = append(recorder.received, event)
			recorder.mu.Unlock()
		}
	}()

	return recorder
}

func (r *eventRecorder) snapshot() []events.Event {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]events.Event(nil), r.received...)
}

func (e *Env) Events() []events.Event {
	return e.events.snapshot()
}

func (e *Env) DoTracked(t *testing.T, req *http.Request) (*Result, Request) {
	t.Helper()

	from := len(e.events.snapshot())
	method, url := req.Method, req.URL.String()
	result := e.Do(t, req)

	var tracked Request

	require.Eventually(t, func() bool {
		var found bool

		tracked, found = findRequest(e.events.snapshot()[from:], method, url)

		return found
	}, eventWait, eventPollTick, "no request events for %s %s", method, url)

	return result, tracked
}

func findRequest(received []events.Event, method, url string) (Request, bool) {
	started := map[uint64]events.RequestStarted{}

	for _, event := range received {
		switch typed := event.(type) {
		case events.RequestStarted:
			started[typed.ID] = typed
		case events.RequestCompleted:
			if typed.Data.Method == method && urlt.URL_String(typed.Data.URL) == url {
				return Request{Started: started[typed.ID], Completed: typed}, true
			}
		}
	}

	return Request{}, false
}
