package server_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/certs"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tracking"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/log"
)

func TestServerRecoversHandlerPanics(t *testing.T) {
	bus := events.NewBus()
	subscription := bus.Subscribe(10)

	panicking := infra.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) error {
		tracking.Claim(request.Context(), events.HandlerMock)

		switch request.URL.Path {
		case "/panic":
			panic("mock exploded")
		case "/panic-after-write":
			writer.WriteHeader(http.StatusOK)
			fmt.Fprint(writer, strings.Repeat("partial", 1024))
			panic("mock exploded")
		case "/abort":
			panic(http.ErrAbortHandler)
		}

		writer.WriteHeader(http.StatusNoContent)

		return nil
	})
	handler := infra.CastToHTTPHandler(panicking)

	port := testutils.GetFreePorts(t, 1)[0]
	instance := server.New(certs.NewManager(afero.NewMemMapFs(), bus).GetCertificate, bus)
	require.NoError(t, instance.Start(t.Context(), []server.Target{{
		Address: hosts.Loopback.Port(port).String(),
		Handler: handler,
	}}))

	defer func() {
		require.NoError(t, instance.Close())
	}()

	fetch := func(t *testing.T, path string) (*http.Response, string, error) {
		t.Helper()

		url := hosts.Loopback.HTTPPort(port).String() + path
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		require.NoError(t, err)
		req.Header.Set(headers.Origin, "http://localhost:3000")

		response, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, "", err
		}

		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)

		return response, string(body), err
	}

	completed := func(t *testing.T) events.RequestCompleted {
		t.Helper()

		next := func() events.Event {
			select {
			case event := <-subscription.Events():
				return event
			case <-time.After(5 * time.Second):
				require.FailNow(t, "no event published")

				return nil
			}
		}

		_, ok := next().(events.RequestStarted)
		require.True(t, ok)

		event, ok := next().(events.RequestCompleted)
		require.True(t, ok)

		return event
	}

	t.Run("answers 500 with CORS headers when nothing was sent", func(t *testing.T) {
		response, body, err := fetch(t, "/panic")
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		assert.Equal(t, "http://localhost:3000", response.Header.Get(headers.AccessControlAllowOrigin))
		assert.Contains(t, body, "handler panicked: mock exploded")

		event := completed(t)
		assert.Equal(t, events.HandlerMock, event.Handler)
		assert.Equal(t, http.StatusInternalServerError, event.Data.Code)
		require.Error(t, event.Err)
		assert.Contains(t, event.Err.Error(), "mock exploded")
	})

	t.Run("aborts the response when it was started", func(t *testing.T) {
		_, _, err := fetch(t, "/panic-after-write")
		require.Error(t, err)

		event := completed(t)
		assert.Equal(t, http.StatusOK, event.Data.Code)
		require.Error(t, event.Err)
		assert.Contains(t, event.Err.Error(), "mock exploded")
	})

	t.Run("passes http.ErrAbortHandler through", func(t *testing.T) {
		_, _, err := fetch(t, "/abort")
		require.Error(t, err)

		event := completed(t)
		assert.NoError(t, event.Err)
	})

	t.Run("keeps serving after a panic", func(t *testing.T) {
		response, _, err := fetch(t, "/ok")
		require.NoError(t, err)

		assert.Equal(t, http.StatusNoContent, response.StatusCode)
		assert.Equal(t, http.StatusNoContent, completed(t).Data.Code)
	})
}

func TestHandleRequestPanicTelemetry(t *testing.T) {
	recorded := testutils.Telemetry(t)

	exploding := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("mock exploded")
	})

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost:8080/api", nil)
	recorder := httptest.NewRecorder()
	server.HandleRequest(server.New(nil, events.NoopPublisher{}), exploding, recorder, request)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)

	ended := recorded.Spans.Ended()
	require.Len(t, ended, 1)
	assert.Equal(t, codes.Error, ended[0].Status().Code)
	assert.Contains(t, ended[0].Status().Description, "handler panicked: mock exploded")

	records := recorded.Records()
	require.Len(t, records, 1)
	assert.Equal(t, log.SeverityError, records[0].Severity)
	assert.Equal(t, "handler panicked", records[0].Message)
	assert.Contains(t, records[0].Attrs["stack"], "server_panic_test.go")
}
