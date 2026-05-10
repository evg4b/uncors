package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeRequest(method, rawURL string) *http.Request {
	u, _ := url.Parse(rawURL)
	req := httptest.NewRequestWithContext(context.Background(), method, rawURL, nil)
	req.URL = u

	return req
}

func makeWriter() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

func TestRequestTracker_Wrap(t *testing.T) {
	t.Run("sends start event with request metadata", func(t *testing.T) {
		tracker := server.NewRequestTracker(mocks.NoopOutput())
		handlerDone := make(chan struct{})

		wrapped := tracker.Wrap(contracts.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) {
			close(handlerDone)
		}))

		go wrapped.ServeHTTP(makeWriter(), makeRequest(http.MethodGet, "http://example.com/path"))

		event := <-tracker.Events()

		assert.False(t, event.Done)
		assert.Equal(t, http.MethodGet, event.Method)
		assert.Equal(t, "/path", event.URL.Path)
		assert.NotZero(t, event.ID)
		assert.NotZero(t, event.StartedAt)

		<-handlerDone
		<-tracker.Events() // drain done event
	})

	t.Run("sends done event after handler returns", func(t *testing.T) {
		tracker := server.NewRequestTracker(mocks.NoopOutput())
		handlerDone := make(chan struct{})
		blocker := make(chan struct{})

		wrapped := tracker.Wrap(contracts.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) {
			<-blocker
			close(handlerDone)
		}))

		go wrapped.ServeHTTP(makeWriter(), makeRequest(http.MethodPost, "http://example.com/api"))

		startEv := <-tracker.Events()
		require.False(t, startEv.Done)

		// handler is still blocked — no done event yet
		select {
		case <-tracker.Events():
			t.Fatal("received done event before handler returned")
		default:
		}

		close(blocker)
		<-handlerDone

		doneEv := <-tracker.Events()
		assert.True(t, doneEv.Done)
		assert.Equal(t, startEv.ID, doneEv.ID)
	})

	t.Run("start event carries correct method and URL", func(t *testing.T) {
		tracker := server.NewRequestTracker(mocks.NoopOutput())

		wrapped := tracker.Wrap(contracts.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) {}))
		go wrapped.ServeHTTP(makeWriter(), makeRequest(http.MethodDelete, "http://host.local/resource?q=1"))

		event := <-tracker.Events()
		<-tracker.Events() // done

		assert.Equal(t, http.MethodDelete, event.Method)
		assert.Equal(t, "/resource", event.URL.Path)
		assert.Equal(t, "q=1", event.URL.RawQuery)
	})

	t.Run("underlying handler is called exactly once", func(t *testing.T) {
		tracker := server.NewRequestTracker(mocks.NoopOutput())
		calls := 0

		wrapped := tracker.Wrap(contracts.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) {
			calls++
		}))

		wrapped.ServeHTTP(makeWriter(), makeRequest(http.MethodGet, "http://example.com/"))
		<-tracker.Events()
		<-tracker.Events()

		assert.Equal(t, 1, calls)
	})

	t.Run("concurrent requests get unique IDs", func(t *testing.T) {
		tracker := server.NewRequestTracker(mocks.NoopOutput())

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
			event := <-tracker.Events()
			assert.False(t, seen[event.ID], "duplicate ID %d", event.ID)
			seen[event.ID] = true
		}

		for range requestCount {
			<-tracker.Events() // drain done events
		}

		assert.Len(t, seen, requestCount)
	})

	t.Run("IDs are monotonically increasing", func(t *testing.T) {
		tracker := server.NewRequestTracker(mocks.NoopOutput())

		wrapped := tracker.Wrap(contracts.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) {}))

		ids := make([]uint64, 0, 5)

		for range 5 {
			wrapped.ServeHTTP(makeWriter(), makeRequest(http.MethodGet, "http://example.com/"))

			ev := <-tracker.Events()
			ids = append(ids, ev.ID)

			<-tracker.Events() // done
		}

		for i := 1; i < len(ids); i++ {
			assert.Greater(t, ids[i], ids[i-1])
		}
	})

	t.Run("logs request with module prefix from PrefixUpdaterKey", func(t *testing.T) {
		var buf strings.Builder

		tracker := server.NewRequestTracker(tui.NewCliOutput(&buf))

		const modulePrefix = "PROXY"

		wrapped := tracker.Wrap(contracts.HandlerFunc(func(_ contracts.ResponseWriter, req *contracts.Request) {
			if updater, ok := req.Context().Value(contracts.PrefixUpdaterKey).(func(string)); ok {
				updater(modulePrefix)
			}
		}))

		wrapped.ServeHTTP(makeWriter(), makeRequest(http.MethodGet, "http://example.com/path"))

		assert.Contains(t, buf.String(), modulePrefix)
		assert.Contains(t, buf.String(), "200")
		assert.Contains(t, buf.String(), "GET")
	})

	t.Run("logs request without prefix when no module is identified", func(t *testing.T) {
		var buf strings.Builder

		tracker := server.NewRequestTracker(tui.NewCliOutput(&buf))

		wrapped := tracker.Wrap(contracts.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) {}))

		wrapped.ServeHTTP(makeWriter(), makeRequest(http.MethodGet, "http://example.com/path"))

		assert.Contains(t, buf.String(), "200")
		assert.Contains(t, buf.String(), "GET")
	})
}
