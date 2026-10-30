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

var handlerNames = [...]string{
	HandlerNone:    "NONE",
	HandlerProxy:   "PROXY",
	HandlerCache:   "CACHE",
	HandlerMock:    "MOCK",
	HandlerScript:  "SCRPT",
	HandlerStatic:  "STATC",
	HandlerOptions: "OPTNS",
}

func (k HandlerKind) String() string {
	if int(k) < len(handlerNames) {
		return handlerNames[k]
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
