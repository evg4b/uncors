package uncorsapp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeRequest(method, rawURL string) *contracts.Request {
	u, _ := url.Parse(rawURL)
	r := httptest.NewRequestWithContext(context.Background(), method, rawURL, nil)
	r.URL = u

	return r
}

func makeWriter() contracts.ResponseWriter {
	return contracts.WrapResponseWriter(httptest.NewRecorder())
}

func TestRequestTracker_Wrap(t *testing.T) {
	t.Run("sends start event with request metadata", func(t *testing.T) {
		tracker := newRequestTracker()
		handlerDone := make(chan struct{})

		wrapped := tracker.Wrap(contracts.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) {
			close(handlerDone)
		}))

		go wrapped.ServeHTTP(makeWriter(), makeRequest(http.MethodGet, "http://example.com/path"))

		event := <-tracker.events

		assert.False(t, event.done)
		assert.Equal(t, http.MethodGet, event.method)
		assert.Equal(t, "/path", event.url.Path)
		assert.NotZero(t, event.id)
		assert.NotZero(t, event.startedAt)

		<-handlerDone
		<-tracker.events // drain done event
	})

	t.Run("sends done event after handler returns", func(t *testing.T) {
		tracker := newRequestTracker()
		handlerDone := make(chan struct{})
		blocker := make(chan struct{})

		wrapped := tracker.Wrap(contracts.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) {
			<-blocker
			close(handlerDone)
		}))

		go wrapped.ServeHTTP(makeWriter(), makeRequest(http.MethodPost, "http://example.com/api"))

		startEv := <-tracker.events
		require.False(t, startEv.done)

		// handler is still blocked — no done event yet
		select {
		case <-tracker.events:
			t.Fatal("received done event before handler returned")
		default:
		}

		close(blocker)
		<-handlerDone

		doneEv := <-tracker.events
		assert.True(t, doneEv.done)
		assert.Equal(t, startEv.id, doneEv.id)
	})

	t.Run("start event carries correct method and URL", func(t *testing.T) {
		tracker := newRequestTracker()

		wrapped := tracker.Wrap(contracts.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) {}))
		go wrapped.ServeHTTP(makeWriter(), makeRequest(http.MethodDelete, "http://host.local/resource?q=1"))

		event := <-tracker.events
		<-tracker.events // done

		assert.Equal(t, http.MethodDelete, event.method)
		assert.Equal(t, "/resource", event.url.Path)
		assert.Equal(t, "q=1", event.url.RawQuery)
	})

	t.Run("underlying handler is called exactly once", func(t *testing.T) {
		tracker := newRequestTracker()
		calls := 0

		wrapped := tracker.Wrap(contracts.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) {
			calls++
		}))

		wrapped.ServeHTTP(makeWriter(), makeRequest(http.MethodGet, "http://example.com/"))
		<-tracker.events
		<-tracker.events

		assert.Equal(t, 1, calls)
	})

	t.Run("concurrent requests get unique IDs", func(t *testing.T) {
		tracker := newRequestTracker()

		const requestCount = 10

		var wg sync.WaitGroup
		wg.Add(requestCount)

		wrapped := tracker.Wrap(contracts.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) {
			wg.Done()
			wg.Wait()
		}))

		for range requestCount {
			go wrapped.ServeHTTP(makeWriter(), makeRequest(http.MethodGet, "http://example.com/"))
		}

		seen := make(map[uint64]bool)

		for range requestCount {
			event := <-tracker.events
			assert.False(t, seen[event.id], "duplicate ID %d", event.id)
			seen[event.id] = true
		}

		for range requestCount {
			<-tracker.events // drain done events
		}

		assert.Len(t, seen, requestCount)
	})

	t.Run("IDs are monotonically increasing", func(t *testing.T) {
		tracker := newRequestTracker()

		wrapped := tracker.Wrap(contracts.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) {}))

		ids := make([]uint64, 0, 5)

		for range 5 {
			wrapped.ServeHTTP(makeWriter(), makeRequest(http.MethodGet, "http://example.com/"))

			ev := <-tracker.events
			ids = append(ids, ev.id)

			<-tracker.events // done
		}

		for i := 1; i < len(ids); i++ {
			assert.Greater(t, ids[i], ids[i-1])
		}
	})
}
