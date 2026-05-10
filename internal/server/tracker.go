package server

import (
	"context"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
)

const requestEventsBufferSize = 1000

type RequestEvent struct {
	ID        uint64
	Method    string
	URL       *url.URL
	StartedAt time.Time
	Prefix    string
	Done      bool
}

type RequestTracker struct {
	events chan RequestEvent
	nextID atomic.Uint64
	output contracts.Output
}

func NewRequestTracker(output contracts.Output) *RequestTracker {
	return &RequestTracker{
		events: make(chan RequestEvent, requestEventsBufferSize),
		output: output,
	}
}

func (t *RequestTracker) Events() <-chan RequestEvent {
	return t.events
}

func (t *RequestTracker) Close() {
	close(t.events)
}

func (t *RequestTracker) Emit(event RequestEvent) {
	select {
	case t.events <- event:
	default:
	}
}

func (t *RequestTracker) Wrap(handler contracts.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		writer := contracts.WrapResponseWriter(w)

		requestID := t.nextID.Add(1)
		select {
		case t.events <- RequestEvent{
			ID:        requestID,
			Method:    req.Method,
			URL:       req.URL,
			StartedAt: time.Now(),
		}:
		default:
		}

		var lastPrefix string

		ctx := context.WithValue(req.Context(), contracts.PrefixUpdaterKey, func(p string) {
			lastPrefix = p
			select {
			case t.events <- RequestEvent{ID: requestID, Prefix: p}:
			default:
			}
		})

		handler.ServeHTTP(writer, req.WithContext(ctx))

		output := t.output
		if lastPrefix != "" {
			output = t.output.NewPrefixOutput(lastPrefix)
		}

		data := helpers.ToRequestData(req, helpers.NormaliseStatusCode(writer.StatusCode()))
		data.Cancelled = ctx.Err() != nil
		output.Request(data)

		select {
		case t.events <- RequestEvent{ID: requestID, Done: true}:
		default:
		}
	})
}
