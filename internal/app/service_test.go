package app_test

import (
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/app"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errLoadFailed = errors.New("config is not valid")

// occupy binds port for the duration of the test so the service cannot.
func occupy(t *testing.T, port int) {
	t.Helper()

	listenConfig := &net.ListenConfig{}

	listener, err := listenConfig.Listen(t.Context(), "tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })
}

func configFor(port int) *config.UncorsConfig {
	return &config.UncorsConfig{
		Mappings: config.Mappings{
			{From: hosts.Localhost.HTTPPort(port), To: hosts.Localhost.HTTP()},
		},
	}
}

func newService(t *testing.T, cfg *config.UncorsConfig, path string, load app.Loader) *app.Service {
	t.Helper()

	container := di.NewContainer()
	service := app.New(container, cfg, path, load)

	t.Cleanup(func() {
		require.NoError(t, service.Shutdown(t.Context()))
		require.NoError(t, service.Close())
		require.NoError(t, container.Close())
	})

	return service
}

func requirePortServing(t *testing.T, port int) {
	t.Helper()

	assert.False(t, testutils.IsPortFree(port), "port %d should be bound", port)
}

// T1: the service is the whole application. Nothing here constructs a TUI.
func TestServiceRunsWithoutAClient(t *testing.T) {
	port := testutils.GetFreePort(t)
	cfg := configFor(port)

	service := newService(t, cfg, "", func() (*config.UncorsConfig, error) {
		return cfg, nil
	})

	require.NoError(t, service.Start(t.Context()))
	requirePortServing(t, port)

	response, err := http.Get("http://localhost:" + strconv.Itoa(port)) //nolint:noctx // liveness probe
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())

	require.NoError(t, service.Shutdown(t.Context()))
	service.Wait()

	assert.Eventually(t, func() bool { return testutils.IsPortFree(port) }, time.Second, 10*time.Millisecond,
		"shutdown must release the listener")
}

func TestServiceStartFailsWhenPortIsTaken(t *testing.T) {
	port := testutils.GetFreePort(t)

	occupy(t, port)

	cfg := configFor(port)
	service := newService(t, cfg, "", func() (*config.UncorsConfig, error) { return cfg, nil })

	require.Error(t, service.Start(t.Context()))
}

// T2 at the service level: a config that fails to load leaves the running
// generation serving, and the failure is reported rather than swallowed.
func TestReloadKeepsGenerationWhenConfigFails(t *testing.T) {
	port := testutils.GetFreePort(t)
	cfg := configFor(port)

	loads := 0
	service := newService(t, cfg, "", func() (*config.UncorsConfig, error) {
		loads++

		return nil, errLoadFailed
	})

	require.NoError(t, service.Start(t.Context()))

	require.NotPanics(t, service.Reload)

	assert.Equal(t, 1, loads)
	requirePortServing(t, port)
	assert.Same(t, cfg, service.Config(), "a failed reload must not swap the active config")
}

func TestReloadMovesToThePortOfTheNewConfig(t *testing.T) {
	first := testutils.GetFreePort(t)
	second := testutils.GetFreePort(t)

	next := configFor(second)
	service := newService(t, configFor(first), "", func() (*config.UncorsConfig, error) {
		return next, nil
	})

	require.NoError(t, service.Start(t.Context()))
	requirePortServing(t, first)

	service.Reload()

	requirePortServing(t, second)
	assert.Same(t, next, service.Config())
	assert.Eventually(t, func() bool { return testutils.IsPortFree(first) }, time.Second, 10*time.Millisecond,
		"the dropped port must be released")
}

// T8: concurrent reloads must be serialised, and none may be lost.
func TestConcurrentReloadsAreSerialised(t *testing.T) {
	const reloaders = 8

	port := testutils.GetFreePort(t)
	cfg := configFor(port)

	var (
		guard    sync.Mutex
		inFlight int
		overlaps int
		loads    int
	)

	service := newService(t, cfg, "", func() (*config.UncorsConfig, error) {
		guard.Lock()
		inFlight++
		loads++

		if inFlight > 1 {
			overlaps++
		}
		guard.Unlock()

		time.Sleep(time.Millisecond)

		guard.Lock()
		inFlight--
		guard.Unlock()

		return cfg, nil
	})

	require.NoError(t, service.Start(t.Context()))

	var waitGroup sync.WaitGroup

	waitGroup.Add(reloaders)

	for range reloaders {
		go func() {
			defer waitGroup.Done()

			service.Reload()
		}()
	}

	waitGroup.Wait()

	guard.Lock()
	defer guard.Unlock()

	assert.Zero(t, overlaps, "reloads must never run concurrently")
	assert.Positive(t, loads, "coalescing must not swallow every request")
	requirePortServing(t, port)
}

func TestServiceWatchesConfigFile(t *testing.T) {
	port := testutils.GetFreePort(t)
	cfg := configFor(port)

	path := filepath.Join(t.TempDir(), "uncors.yaml")
	require.NoError(t, os.WriteFile(path, []byte("mappings: []"), 0o600))

	reloaded := make(chan struct{}, 1)
	service := newService(t, cfg, path, func() (*config.UncorsConfig, error) {
		select {
		case reloaded <- struct{}{}:
		default:
		}

		return cfg, nil
	})

	require.NoError(t, service.Start(t.Context()))

	require.NoError(t, os.WriteFile(path, []byte("proxy: \"\""), 0o600))

	select {
	case <-reloaded:
	case <-time.After(2 * time.Second):
		t.Fatal("a config file change must trigger a reload")
	}
}

func TestServiceShutdownIsIdempotent(t *testing.T) {
	port := testutils.GetFreePort(t)
	cfg := configFor(port)

	service := newService(t, cfg, "", func() (*config.UncorsConfig, error) { return cfg, nil })

	require.NoError(t, service.Start(t.Context()))

	require.NoError(t, service.Shutdown(t.Context()))
	require.NoError(t, service.Shutdown(t.Context()))

	assert.Error(t, service.Context().Err(), "shutdown must cancel the service context")
}
