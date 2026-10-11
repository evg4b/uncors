package cli

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var errFrontend = errors.New("frontend failed")

const configPath = "/config.yaml"

// supervisedConfig writes a one-mapping config to a fresh file system and
// returns everything supervise needs to serve it.
func supervisedConfig(t *testing.T) (Env, *config.UncorsConfig) {
	t.Helper()

	port := testutils.GetFreePort(t)
	uncorsConfig := &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Localhost.HTTPPort(port),
			To:   hosts.Localhost.HTTP(),
		}},
		CacheConfig: config.CacheConfig{
			ExpirationTime: config.DefaultExpirationTime,
			MaxSize:        config.DefaultMaxSize,
			Methods:        []string{http.MethodGet},
		},
	}

	fs := afero.NewMemMapFs()
	data, err := yaml.Marshal(uncorsConfig)
	require.NoError(t, err)
	require.NoError(t, afero.WriteFile(fs, configPath, data, 0o600))

	env := Env{Fs: fs, Stdout: io.Discard, Args: []string{"-c", configPath}, Version: "x.x.x"}

	return env, uncorsConfig
}

// collect returns every event until the subscription closes.
func collect(subscription *events.Subscription) []events.Event {
	var received []events.Event
	for event := range subscription.Events() {
		received = append(received, event)
	}

	return received
}

func eventTypes(received []events.Event) []string {
	types := make([]string, 0, len(received))
	for _, event := range received {
		switch event.(type) {
		case events.ServerStarted:
			types = append(types, "started")
		case events.ServerRestarted:
			types = append(types, "restarted")
		case events.ServerStopped:
			types = append(types, "stopped")
		case events.ReloadFailed:
			types = append(types, "reload-failed")
		}
	}

	return types
}

// waitFor reads events into received until one of type T arrives. It runs on
// the frontend goroutine, so it reports a failure instead of stopping the test.
func waitFor[T events.Event](t *testing.T, subscription *events.Subscription, received *[]events.Event) bool {
	t.Helper()

	for event := range subscription.Events() {
		*received = append(*received, event)
		if _, ok := event.(T); ok {
			return true
		}
	}

	t.Errorf("subscription closed before %T arrived", *new(T))

	return false
}

func TestSupervise(t *testing.T) {
	t.Run("restarts on a command and stops on shutdown", func(t *testing.T) {
		env, uncorsConfig := supervisedConfig(t)

		var received []events.Event

		frontend := func(subscription *events.Subscription, commands chan<- events.Command) error {
			if !waitFor[events.ServerStarted](t, subscription, &received) {
				return nil
			}

			commands <- events.Restart{}

			if !waitFor[events.ServerRestarted](t, subscription, &received) {
				return nil
			}

			commands <- events.Shutdown{}

			received = append(received, collect(subscription)...)

			return nil
		}

		require.NoError(t, supervise(t.Context(), env, uncorsConfig, "", frontend))
		assert.Equal(t, []string{"started", "restarted", "stopped"}, eventTypes(received))
	})

	t.Run("keeps serving when the reloaded config is invalid", func(t *testing.T) {
		env, uncorsConfig := supervisedConfig(t)
		port := uncorsConfig.Mappings[0].From.Port

		var received []events.Event

		frontend := func(subscription *events.Subscription, commands chan<- events.Command) error {
			if !waitFor[events.ServerStarted](t, subscription, &received) {
				return nil
			}

			err := afero.WriteFile(env.Fs, configPath, []byte("mappings: []\n"), 0o600)
			if err != nil {
				t.Error(err)

				return nil
			}

			commands <- events.Restart{}

			if !waitFor[events.ReloadFailed](t, subscription, &received) {
				return nil
			}

			conn, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", net.JoinHostPort("127.0.0.1", port))
			if assert.NoError(t, err, "the previous config must keep serving") {
				_ = conn.Close()
			}

			commands <- events.Shutdown{}

			received = append(received, collect(subscription)...)

			return nil
		}

		require.NoError(t, supervise(t.Context(), env, uncorsConfig, "", frontend))
		assert.Equal(t, []string{"started", "reload-failed", "stopped"}, eventTypes(received))
	})

	t.Run("shuts down when the frontend exits", func(t *testing.T) {
		env, uncorsConfig := supervisedConfig(t)

		frontend := func(subscription *events.Subscription, _ chan<- events.Command) error {
			var received []events.Event
			if !waitFor[events.ServerStarted](t, subscription, &received) {
				return nil
			}

			return errFrontend
		}

		err := supervise(t.Context(), env, uncorsConfig, "", frontend)
		require.ErrorIs(t, err, errFrontend)

		port, _ := strconv.Atoi(uncorsConfig.Mappings[0].From.Port)

		assert.Eventually(t, func() bool {
			conn, dialErr := (&net.Dialer{}).DialContext(t.Context(), "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
			if dialErr != nil {
				return true
			}

			_ = conn.Close()

			return false
		}, time.Second, 20*time.Millisecond, "the listener must be closed")
	})

	t.Run("stops on context cancellation", func(t *testing.T) {
		env, uncorsConfig := supervisedConfig(t)
		ctx, cancel := context.WithCancel(t.Context())

		var received []events.Event

		frontend := func(subscription *events.Subscription, _ chan<- events.Command) error {
			if !waitFor[events.ServerStarted](t, subscription, &received) {
				return nil
			}

			cancel()

			received = append(received, collect(subscription)...)

			return nil
		}

		require.NoError(t, supervise(ctx, env, uncorsConfig, "", frontend))
		assert.Equal(t, []string{"started", "stopped"}, eventTypes(received))
	})

	t.Run("reports a failed start and stops the frontend", func(t *testing.T) {
		env, uncorsConfig := supervisedConfig(t)

		listener, err := (&net.ListenConfig{}).Listen(
			t.Context(), "tcp4", net.JoinHostPort("127.0.0.1", uncorsConfig.Mappings[0].From.Port),
		)
		require.NoError(t, err)

		defer listener.Close()

		var received []events.Event

		frontend := func(subscription *events.Subscription, _ chan<- events.Command) error {
			received = collect(subscription)

			return nil
		}

		require.Error(t, supervise(t.Context(), env, uncorsConfig, "", frontend))
		assert.Equal(t, []string{"stopped"}, eventTypes(received))
	})
}

type recordingWriter struct {
	written []byte
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	w.written = append(w.written, p...)

	return len(p), nil
}

func TestOnSignal(t *testing.T) {
	t.Run("moves past the echoed ^C in headless mode", func(t *testing.T) {
		output := &recordingWriter{}
		supervisor := &supervisor{env: Env{Stdout: output}, logger: Env{}.logger()}

		supervisor.onSignal(syscall.SIGINT)

		assert.Equal(t, "\n", string(output.written))
	})

	t.Run("writes nothing in interactive mode", func(t *testing.T) {
		output := &recordingWriter{}
		supervisor := &supervisor{env: Env{Stdout: output}, logger: Env{}.logger(), interactive: true}

		supervisor.onSignal(syscall.SIGINT)

		assert.Empty(t, output.written)
	})

	t.Run("writes nothing for other signals", func(t *testing.T) {
		output := &recordingWriter{}
		supervisor := &supervisor{env: Env{Stdout: output}, logger: Env{}.logger()}

		supervisor.onSignal(syscall.SIGTERM)

		assert.Empty(t, output.written)
	})
}

func TestEnvLogger(t *testing.T) {
	assert.NotNil(t, Env{}.logger(), "a missing logger discards records")

	logger, _ := testutils.NewBufferLogger()
	assert.Same(t, logger, Env{Logger: logger}.logger())
}
