package testutils

import (
	"sync"

	"github.com/evg4b/uncors/internal/events"
)

// EventRecorder is an events.Publisher that keeps every event it receives.
type EventRecorder struct {
	mu     sync.Mutex
	events []events.Event
}

func (r *EventRecorder) Publish(event events.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = append(r.events, event)
}

func (r *EventRecorder) Notices() []events.Notice {
	r.mu.Lock()
	defer r.mu.Unlock()

	var notices []events.Notice

	for _, event := range r.events {
		if notice, ok := event.(events.Notice); ok {
			notices = append(notices, notice)
		}
	}

	return notices
}
