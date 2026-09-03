package app_test

import (
	"sync"
	"testing"

	"github.com/evg4b/uncors/internal/app"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func drain(t *testing.T, service *app.Service) []app.Event {
	t.Helper()

	events := make([]app.Event, 0)

	for {
		select {
		case event := <-service.Events():
			events = append(events, event)
		default:
			return events
		}
	}
}

func statesOf(events []app.Event) []app.LifecycleState {
	states := make([]app.LifecycleState, 0, len(events))

	for _, event := range events {
		if lifecycle, ok := event.(app.LifecycleEvent); ok {
			states = append(states, lifecycle.State)
		}
	}

	return states
}

func TestServiceEmitsLifecycle(t *testing.T) {
	t.Run("a successful start reports starting then started with its mappings", func(t *testing.T) {
		port := testutils.GetFreePort(t)
		cfg := configFor(port)

		service := newService(t, cfg, "", func() (*config.UncorsConfig, error) { return cfg, nil })

		require.NoError(t, service.Start(t.Context()))

		events := drain(t, service)
		assert.Equal(t, []app.LifecycleState{app.StateStarting, app.StateStarted}, statesOf(events))

		started, ok := events[1].(app.LifecycleEvent)
		require.True(t, ok)
		assert.Equal(t, cfg.Mappings, started.Mappings,
			"the event must describe the generation without the client asking the service")
	})

	t.Run("a failed start reports the error", func(t *testing.T) {
		port := testutils.GetFreePort(t)

		occupy(t, port)

		cfg := configFor(port)
		service := newService(t, cfg, "", func() (*config.UncorsConfig, error) { return cfg, nil })

		require.Error(t, service.Start(t.Context()))
		assert.Equal(t, app.StateStartFailed, service.Status().State)
		require.Error(t, service.Status().Err)
	})

	t.Run("a failed reload reports the error and keeps the old mappings", func(t *testing.T) {
		port := testutils.GetFreePort(t)
		cfg := configFor(port)

		service := newService(t, cfg, "", func() (*config.UncorsConfig, error) { return nil, errLoadFailed })

		require.NoError(t, service.Start(t.Context()))

		service.Reload()

		assert.Equal(t, app.StateReloadFailed, service.Status().State)
		require.ErrorIs(t, service.Status().Err, errLoadFailed)
		assert.Same(t, cfg, service.Config())
	})
}

// Status must stay truthful even when nobody is draining the stream, because
// that is the whole reason lifecycle is state rather than only a notification.
func TestStatusSurvivesAnUndrainedStream(t *testing.T) {
	port := testutils.GetFreePort(t)
	cfg := configFor(port)

	service := newService(t, cfg, "", func() (*config.UncorsConfig, error) { return cfg, nil })

	require.NoError(t, service.Start(t.Context()))

	for range 3000 {
		service.Reload()
	}

	assert.Positive(t, service.DroppedEvents(), "an undrained stream must drop and count, not block")
	assert.Equal(t, app.StateReloaded, service.Status().State)
}

// Emitting must never block the caller, whatever the consumer is doing.
func TestEmittingNeverBlocks(t *testing.T) {
	port := testutils.GetFreePort(t)
	cfg := configFor(port)

	service := newService(t, cfg, "", func() (*config.UncorsConfig, error) { return cfg, nil })

	require.NoError(t, service.Start(t.Context()))

	var waitGroup sync.WaitGroup

	waitGroup.Go(func() {
		for range 500 {
			service.Reload()
		}
	})

	done := make(chan struct{})

	go func() {
		waitGroup.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-t.Context().Done():
		t.Fatal("emitting blocked")
	}
}

func TestClosedServiceStopsDelivery(t *testing.T) {
	cfg := configFor(testutils.GetFreePort(t))

	container := di.NewContainer()
	service := app.New(container, cfg, "", func() (*config.UncorsConfig, error) { return cfg, nil })

	require.NoError(t, service.Close())

	_, open := <-service.Events()
	assert.False(t, open, "Close must close the event stream so consumers terminate")

	require.NoError(t, container.Close())
}
