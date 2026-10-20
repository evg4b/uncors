package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"gopkg.in/yaml.v3"
)

var errFrontend = errors.New("frontend failed")

const configPath = "/config.yaml"

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

func TestSuperviseRestartsAndStops(t *testing.T) {
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
}

func TestSuperviseKeepsServingOnInvalidReload(t *testing.T) {
	spans := testutils.Telemetry(t).Spans
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
	assert.Contains(t, received, events.Notice{
		Level: events.NoticeInfo,
		Text:  "Tracing is enabled, exporting to " + telemetry.Endpoint(),
	})

	ended := spans.Ended()
	reload := slices.IndexFunc(ended, func(span sdktrace.ReadOnlySpan) bool {
		return span.Name() == "supervisor.reload"
	})
	require.GreaterOrEqual(t, reload, 0)
	assert.Equal(t, codes.Error, ended[reload].Status().Code)

	load := slices.IndexFunc(ended, func(span sdktrace.ReadOnlySpan) bool {
		return span.Name() == "config.load" && span.Parent().SpanID() == ended[reload].SpanContext().SpanID()
	})
	require.GreaterOrEqual(t, load, 0, "config.load nests under supervisor.reload")
	assert.Equal(t, codes.Error, ended[load].Status().Code)
}

func TestSuperviseStopsWhenFrontendExits(t *testing.T) {
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

	port := uncorsConfig.Mappings[0].From.Port

	assert.Eventually(t, func() bool {
		conn, dialErr := (&net.Dialer{}).DialContext(t.Context(), "tcp", net.JoinHostPort("127.0.0.1", port))
		if dialErr != nil {
			return true
		}

		_ = conn.Close()

		return false
	}, time.Second, 20*time.Millisecond, "the listener must be closed")
}

func TestSuperviseStopsOnContextCancellation(t *testing.T) {
	env, uncorsConfig := supervisedConfig(t)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

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
}

func TestSuperviseReportsFailedStart(t *testing.T) {
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
}

type blockingFs struct {
	afero.Fs

	armed   atomic.Bool
	entered chan struct{}
	release chan struct{}
}

func (f *blockingFs) Open(name string) (afero.File, error) {
	if name == configPath && f.armed.CompareAndSwap(true, false) {
		close(f.entered)
		<-f.release
	}

	return f.Fs.Open(name)
}

func TestSuperviseDoesNotWaitForReload(t *testing.T) {
	stops := map[string]func(t *testing.T, commands chan<- events.Command){
		"shutdown command": func(_ *testing.T, commands chan<- events.Command) {
			commands <- events.Shutdown{}
		},
		"SIGINT": func(t *testing.T, _ chan<- events.Command) {
			t.Helper()

			assert.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGINT))
		},
	}

	for name, stop := range stops {
		t.Run(name, func(t *testing.T) {
			env, uncorsConfig := supervisedConfig(t)
			fs := &blockingFs{Fs: env.Fs, entered: make(chan struct{}), release: make(chan struct{})}
			env.Fs = fs

			defer close(fs.release)

			var received []events.Event

			frontend := func(subscription *events.Subscription, commands chan<- events.Command) error {
				if !waitFor[events.ServerStarted](t, subscription, &received) {
					return errFrontend
				}

				fs.armed.Store(true)

				commands <- events.Restart{}

				<-fs.entered

				stop(t, commands)

				received = append(received, collect(subscription)...)

				return nil
			}

			done := make(chan error, 1)

			go func() {
				done <- supervise(t.Context(), env, uncorsConfig, "", frontend)
			}()

			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("supervise waited for the reload to finish")
			}

			assert.Equal(t, []string{"started", "stopped"}, eventTypes(received))

			port, err := strconv.Atoi(uncorsConfig.Mappings[0].From.Port)
			require.NoError(t, err)
			assert.True(t, testutils.IsPortFree(port))
		})
	}
}

func TestOnSignal(t *testing.T) {
	t.Run("moves past the echoed ^C in headless mode", func(t *testing.T) {
		output := &bytes.Buffer{}
		supervisor := &supervisor{env: Env{Stdout: output}}

		supervisor.onSignal(syscall.SIGINT)

		assert.Equal(t, "\n", output.String())
	})

	t.Run("writes nothing in interactive mode", func(t *testing.T) {
		output := &bytes.Buffer{}
		supervisor := &supervisor{env: Env{Stdout: output}, interactive: true}

		supervisor.onSignal(syscall.SIGINT)

		assert.Empty(t, output.String())
	})

	t.Run("writes nothing for other signals", func(t *testing.T) {
		output := &bytes.Buffer{}
		supervisor := &supervisor{env: Env{Stdout: output}}

		supervisor.onSignal(syscall.SIGTERM)

		assert.Empty(t, output.String())
	})
}
