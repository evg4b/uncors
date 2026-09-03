package app

import (
	"sync"
	"sync/atomic"

	"github.com/evg4b/uncors/internal/config"
)

const eventsBufferSize = 1000

// Event is something the service reports to whoever is presenting it. The set
// is deliberately small: it covers what the application already communicated,
// and nothing more.
type Event interface {
	isEvent()
}

// LifecycleState is the state of the server as the service understands it.
type LifecycleState int

const (
	StateStarting LifecycleState = iota
	StateStarted
	StateStartFailed
	StateReloading
	StateReloaded
	StateReloadFailed
	StateStopping
	StateStopped
)

// LifecycleEvent reports a change of server state. Mappings carries the
// configuration of the generation the state refers to, so a presenter never
// has to reach back into the service to describe what is serving.
type LifecycleEvent struct {
	State    LifecycleState
	Mappings config.Mappings
	Err      error

	// Interrupted marks a stop caused by SIGINT, where the terminal has already
	// echoed "^C" and the presenter may need to move past it.
	Interrupted bool
}

func (LifecycleEvent) isEvent() {}

// Level classifies a LogEvent for presentation. It carries no rendering.
type Level int

const (
	LevelInfo Level = iota
	LevelWarn
	LevelError
)

// LogEvent is a message from the service addressed to the user.
type LogEvent struct {
	Level   Level
	Prefix  string
	Message string
}

func (LogEvent) isEvent() {}

// Status is the latest lifecycle state, always readable regardless of whether
// the notification for it was delivered.
type Status struct {
	State    LifecycleState
	Mappings config.Mappings
	Err      error
}

// emitter fans service events out to the single presenting client.
//
// Log events are dropped when the client cannot keep up, and counted, exactly
// as request activity is: presentation must never be able to stall the
// service. Lifecycle notifications are dropped under the same pressure, which
// is safe only because the latest state is also recorded in status - a client
// that misses a notification can still read the truth.
type emitter struct {
	events chan Event

	mu     sync.RWMutex
	status Status

	// sendMu guards closed together with the send itself, so Close can never
	// race a send onto an already-closed channel.
	sendMu  sync.RWMutex
	closed  bool
	dropped atomic.Uint64
}

func newEmitter() *emitter {
	return &emitter{events: make(chan Event, eventsBufferSize)}
}

func (e *emitter) Events() <-chan Event {
	return e.events
}

func (e *emitter) Status() Status {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.status
}

func (e *emitter) Dropped() uint64 {
	return e.dropped.Load()
}

func (e *emitter) EmitLifecycle(event LifecycleEvent) {
	e.mu.Lock()
	e.status = Status{State: event.State, Mappings: event.Mappings, Err: event.Err}
	e.mu.Unlock()

	e.send(event)
}

func (e *emitter) EmitLog(level Level, message string) {
	e.send(LogEvent{Level: level, Message: message})
}

// Close stops delivery. It is safe to call concurrently with an emit and safe
// to call more than once.
func (e *emitter) Close() {
	e.sendMu.Lock()
	defer e.sendMu.Unlock()

	if e.closed {
		return
	}

	e.closed = true

	close(e.events)
}

func (e *emitter) send(event Event) {
	e.sendMu.RLock()
	defer e.sendMu.RUnlock()

	if e.closed {
		e.dropped.Add(1)

		return
	}

	select {
	case e.events <- event:
	default:
		e.dropped.Add(1)
	}
}
