package testutils

import (
	"sync"
	"testing"
	"time"
)

const waitClosedTimeout = 5 * time.Second

type QueueEvents struct {
	mutex  sync.Mutex
	events []string
}

func (q *QueueEvents) Track(event string) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.events = append(q.events, event)
}

func (q *QueueEvents) List() []string {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	cloned := make([]string, len(q.events))
	copy(cloned, q.events)

	return cloned
}

func WaitClosed[T any](t *testing.T, channel <-chan T) {
	t.Helper()

	select {
	case <-channel:
	case <-time.After(waitClosedTimeout):
		t.Fatalf("channel was not closed within %s", waitClosedTimeout)
	}
}
