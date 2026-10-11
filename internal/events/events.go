package events

import (
	"net/url"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
)

type Event interface {
	isEvent()
}

type HandlerKind uint8

const (
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

type Rewrite struct {
	From string
	To   string
}

type RequestStarted struct {
	ID      uint64
	Method  string
	URL     *url.URL
	Handler HandlerKind
	Rewrite *Rewrite
	At      time.Time
}

type RequestCompleted struct {
	ID       uint64
	Handler  HandlerKind
	Rewrite  *Rewrite
	Data     *contracts.RequestData
	Duration time.Duration
	Err      error
}

func (RequestStarted) isEvent() {
	// Seals Event: only types in this package implement it.
}

func (RequestCompleted) isEvent() {
	// Seals Event: only types in this package implement it.
}
