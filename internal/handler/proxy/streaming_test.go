package proxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const flushWait = 5 * time.Second

type flushRecorder struct {
	*httptest.ResponseRecorder

	flushed chan string
}

func newFlushRecorder() *flushRecorder {
	return &flushRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		flushed:          make(chan string, 64),
	}
}

func (r *flushRecorder) Flush() {
	r.flushed <- r.Body.String()
}

func streamingHandler(response *http.Response) *proxy.Handler {
	return proxy.NewProxyHandler(
		proxy.WithHTTPClient(testutils.NewTestClient(func(req *http.Request) *http.Response {
			response.Request = req

			return response
		})),
		proxy.WithURLReplacerFactory(urlreplacer.NewURLReplacerFactory(config.Mappings{
			{From: hosts.Parse("http://premium.local.com"), To: hosts.Parse("https://premium.api.com")},
		})),
	)
}

func TestProxyHandlerStreaming(t *testing.T) {
	tests := []struct {
		name          string
		contentType   string
		contentLength int64
	}{
		{name: "event stream", contentType: "text/event-stream; charset=utf-8", contentLength: 1024},
		{name: "event stream in upper case", contentType: "Text/Event-Stream", contentLength: 1024},
		{name: "body of unknown length", contentType: "application/octet-stream", contentLength: -1},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			upstream, upstreamWriter := io.Pipe()

			t.Cleanup(func() { _ = upstreamWriter.Close() })

			handler := streamingHandler(&http.Response{
				StatusCode:    http.StatusOK,
				Header:        http.Header{headers.ContentType: {testCase.contentType}},
				Body:          upstream,
				ContentLength: testCase.contentLength,
			})

			recorder := newFlushRecorder()
			request := premiumRequest(t, http.MethodGet, nil)
			done := make(chan error, 1)

			go func() {
				done <- handler.ServeHTTP(server.NewResponseRecorder(recorder), request)
			}()

			_, err := io.WriteString(upstreamWriter, "data: 1\n\n")
			require.NoError(t, err)

			waitForFlushed(t, recorder.flushed, "data: 1\n\n")

			_, err = io.WriteString(upstreamWriter, "data: 2\n\n")
			require.NoError(t, err)

			waitForFlushed(t, recorder.flushed, "data: 1\n\ndata: 2\n\n")

			require.NoError(t, upstreamWriter.Close())
			require.NoError(t, <-done)
		})
	}

	t.Run("does not flush a body of known length", func(t *testing.T) {
		const body = `{"id":1}`

		handler := streamingHandler(&http.Response{
			StatusCode:    http.StatusOK,
			Header:        http.Header{headers.ContentType: {"application/json"}},
			Body:          io.NopCloser(strings.NewReader(body)),
			ContentLength: int64(len(body)),
		})

		recorder := newFlushRecorder()

		require.NoError(t, handler.ServeHTTP(server.NewResponseRecorder(recorder), premiumRequest(t, http.MethodGet, nil)))

		assert.Equal(t, body, recorder.Body.String())
		assert.Empty(t, recorder.flushed)
	})
}

func waitForFlushed(t *testing.T, flushed <-chan string, expected string) {
	t.Helper()

	timeout := time.After(flushWait)

	for {
		select {
		case body := <-flushed:
			if body == expected {
				return
			}
		case <-timeout:
			t.Fatalf("%q was not flushed within %s", expected, flushWait)
		}
	}
}
