package uncorsapp

import (
	"context"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
)

const requestEventsBufferSize = 1000

type requestEvent struct {
	id        uint64
	method    string
	url       *url.URL
	startedAt time.Time
	prefix    string
	done      bool
}

type RequestTracker struct {
	events chan requestEvent
	nextID atomic.Uint64
	output contracts.Output
}

func NewRequestTracker(output contracts.Output) *RequestTracker {
	return &RequestTracker{
		events: make(chan requestEvent, requestEventsBufferSize),
		output: output,
	}
}

func (t *RequestTracker) Wrap(handler contracts.Handler) contracts.Handler {
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

		var lastPrefix string

		ctx := context.WithValue(request.Context(), contracts.PrefixUpdaterKey, func(p string) {
			lastPrefix = p
			select {
			case t.events <- requestEvent{id: requestID, prefix: p}:
			default:
			}
		})

		handler.ServeHTTP(writer, request.WithContext(ctx))

		output := t.output
		if lastPrefix != "" {
			output = t.output.NewPrefixOutput(lastPrefix)
		}

		output.Request(helpers.ToRequestData(request, helpers.NormaliseStatusCode(writer.StatusCode())))

		select {
		case t.events <- requestEvent{id: requestID, done: true}:
		default:
		}
	})
}
