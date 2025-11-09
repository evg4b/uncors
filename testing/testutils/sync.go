package testutils

import "sync"

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
