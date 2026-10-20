//go:build integration

package proxy_test

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const firstEventWait = 5 * time.Second

type eventStream struct {
	*integration.Backend

	mu      sync.Mutex
	release chan struct{}
}

func newEventStream(t *testing.T) *eventStream {
	t.Helper()

	stream := &eventStream{}
	stream.Backend = integration.NewBackend(t, func(writer http.ResponseWriter, request *http.Request) {
		release := stream.next()

		writer.Header().Set("Content-Type", "text/event-stream")
		writer.WriteHeader(http.StatusOK)

		_, err := io.WriteString(writer, "data: 1\n\n")
		assert.NoError(t, err)
		assert.NoError(t, http.NewResponseController(writer).Flush())

		select {
		case <-release:
		case <-request.Context().Done():
			return
		}

		_, err = io.WriteString(writer, "data: 2\n\n")
		assert.NoError(t, err)
	})

	return stream
}

func (s *eventStream) next() chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.release
}

func (s *eventStream) arm() func() {
	release := make(chan struct{})

	s.mu.Lock()
	s.release = release
	s.mu.Unlock()

	var once sync.Once

	return func() { once.Do(func() { close(release) }) }
}

func TestProxyStreaming(t *testing.T) {
	stream := newEventStream(t)
	env := integration.New(t, stream.Backend, &config.UncorsConfig{
		CacheConfig: config.CacheConfig{
			Methods:        []string{http.MethodGet},
			MaxSize:        10 * 1024 * 1024,
			ExpirationTime: 5 * time.Minute,
		},
		Mappings: config.Mappings{
			{From: hosts.Parse("https://events.local"), To: stream.AsHost()},
			{From: hosts.Parse("https://cached-events.local"), To: stream.AsHost(), Cache: config.CacheGlobs{"/**"}},
		},
	})

	for _, host := range []string{"events.local", "cached-events.local"} {
		t.Run("delivers the first event before the upstream finishes via "+host, func(t *testing.T) {
			release := stream.arm()
			t.Cleanup(release)

			ctx, cancel := context.WithTimeout(t.Context(), firstEventWait)
			defer cancel()

			request, err := http.NewRequestWithContext(ctx, http.MethodGet, env.URL(host, "/events"), http.NoBody)
			require.NoError(t, err)

			response, err := env.Client.Do(request)
			require.NoError(t, err)

			defer response.Body.Close()

			assert.Equal(t, "text/event-stream", response.Header.Get("Content-Type"))

			reader := bufio.NewReader(response.Body)

			event, err := reader.ReadString('\n')
			require.NoError(t, err)
			assert.Equal(t, "data: 1\n", event)

			release()

			rest, err := io.ReadAll(reader)
			require.NoError(t, err)
			assert.Equal(t, "\ndata: 2\n\n", string(rest))
		})
	}
}
