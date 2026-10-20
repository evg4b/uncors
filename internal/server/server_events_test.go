package server_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/certs"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tracking"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errMockFailed = errors.New("mock failed")

func TestServerPublishesRequestEvents(t *testing.T) {
	bus := events.NewBus()
	subscription := bus.Subscribe(10)

	claiming := infra.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) error {
		tracking.Claim(request.Context(), events.HandlerMock)

		if request.URL.Path == "/fail" {
			return errMockFailed
		}

		writer.WriteHeader(http.StatusAccepted)

		return nil
	})
	handler := infra.CastToHTTPHandler(claiming)

	port := testutils.GetFreePorts(t, 1)[0]
	instance := server.New(certs.NewManager(afero.NewMemMapFs(), bus).GetCertificate, bus)
	require.NoError(t, instance.Start(t.Context(), []server.Target{{
		Address: hosts.Loopback.Port(port).String(),
		Handler: handler,
	}}))

	defer func() {
		require.NoError(t, instance.Close())
	}()

	request := func(path string) (events.RequestStarted, events.RequestCompleted) {
		t.Helper()

		url := hosts.Loopback.HTTPPort(port).String() + path
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
		require.NoError(t, err)

		response, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())

		started, ok := (<-subscription.Events()).(events.RequestStarted)
		require.True(t, ok)

		completed, ok := (<-subscription.Events()).(events.RequestCompleted)
		require.True(t, ok)

		return started, completed
	}

	t.Run("reports the claiming handler and the response", func(t *testing.T) {
		started, completed := request("/ok")

		assert.Equal(t, started.ID, completed.ID)
		assert.Equal(t, http.MethodGet, started.Method)
		assert.Equal(t, "/ok", started.URL.Path)
		assert.Equal(t, events.HandlerMock, started.Handler)
		assert.Equal(t, events.HandlerMock, completed.Handler)
		assert.Equal(t, http.StatusAccepted, completed.Data.Code)
		assert.NoError(t, completed.Err)
	})

	t.Run("reports the handler error behind an error page", func(t *testing.T) {
		_, completed := request("/fail")

		assert.Equal(t, http.StatusInternalServerError, completed.Data.Code)
		assert.ErrorIs(t, completed.Err, errMockFailed)
	})

	t.Run("gives every request its own id", func(t *testing.T) {
		first, _ := request("/ok")
		second, _ := request("/ok")

		assert.NotEqual(t, first.ID, second.ID)
	})
}
