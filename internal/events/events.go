// Package events defines what the proxy core reports to the outside world and
// the bus that carries it. The core publishes; frontends (the interactive UI,
// the headless printer, tests) subscribe. Nothing here knows how an event is
// rendered.
package events

import (
	"net/url"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
)

// Event is anything published on the Bus.
type Event interface {
	isEvent()
}

// HandlerKind identifies the component that produced a response.
type HandlerKind uint8

const (
	// HandlerNone means no component took the request, for example because
	// its host is not mapped.
	HandlerNone HandlerKind = iota
	HandlerProxy
	HandlerCache
	HandlerMock
	HandlerScript
	HandlerStatic
	HandlerOptions
)

var handlerNames = map[HandlerKind]string{
	HandlerNone:    "NONE",
	HandlerProxy:   "PROXY",
	HandlerCache:   "CACHE",
	HandlerMock:    "MOCK",
	HandlerScript:  "SCRIPT",
	HandlerStatic:  "STATIC",
	HandlerOptions: "OPTIONS",
}

func (k HandlerKind) String() string {
	if name, ok := handlerNames[k]; ok {
		return name
	}

	return "UNKNOWN"
}

// Rewrite records that a rewrite rule changed the request path before it was
// handled. To includes the host when the rule also replaced it.
type Rewrite struct {
	From string
	To   string
}

// RequestStarted is published once per request, as soon as the component
// that will produce the response is known.
type RequestStarted struct {
	ID      uint64
	Method  string
	URL     *url.URL
	Handler HandlerKind
	Rewrite *Rewrite
	At      time.Time
}

// RequestCompleted is published once per request, after RequestStarted, when
// the response has been written. Err is the error the handler failed with, if
// any; the client received an error page for it.
type RequestCompleted struct {
	ID       uint64
	Handler  HandlerKind
	Rewrite  *Rewrite
	Data     *contracts.RequestData
	Duration time.Duration
	Err      error
}

func (RequestStarted) isEvent()   {}
func (RequestCompleted) isEvent() {}
