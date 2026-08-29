package uncors_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errBrokenConfig = errors.New("mappings[0].from is not a valid host")

func TestReloaderReload(t *testing.T) {
	t.Run("keeps serving the previous config when loading fails", func(t *testing.T) {
		container := di.NewContainer(di.WithVersion(version))
		defer testutils.Close(t, container)

		app := uncors.CreateUncors(container)

		targetServer := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, "OK")
		}))
		defer targetServer.Close()

		port := testutils.GetFreePort(t)

		require.NoError(t, app.Start(t.Context(), &config.UncorsConfig{
			Mappings: config.Mappings{
				{From: hosts.Loopback.HTTPPort(port), To: hosts.Parse(targetServer.URL)},
			},
		}))

		defer app.Close()

		reloader := uncors.NewReloader(app, container.CliOutput(), func() (*config.UncorsConfig, error) {
			return nil, errBrokenConfig
		}, "")

		err := reloader.Reload(t.Context())

		require.ErrorIs(t, err, errBrokenConfig)
		assert.Equal(t, "OK", get(t, hosts.Loopback.HTTPPort(port).String()))
	})

	t.Run("reports a nil configuration instead of panicking", func(t *testing.T) {
		container := di.NewContainer(di.WithVersion(version))
		defer testutils.Close(t, container)

		app := uncors.CreateUncors(container)
		reloader := uncors.NewReloader(app, container.CliOutput(), func() (*config.UncorsConfig, error) {
			return nil, nil //nolint:nilnil // exactly the contract violation under test
		}, "")

		require.Error(t, reloader.Reload(t.Context()))
	})

	t.Run("applies the new configuration", func(t *testing.T) {
		container := di.NewContainer(di.WithVersion(version))
		defer testutils.Close(t, container)

		app := uncors.CreateUncors(container)

		first := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, "first")
		}))
		defer first.Close()

		second := testutils.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, "second")
		}))
		defer second.Close()

		port := testutils.GetFreePort(t)

		require.NoError(t, app.Start(t.Context(), &config.UncorsConfig{
			Mappings: config.Mappings{
				{From: hosts.Loopback.HTTPPort(port), To: hosts.Parse(first.URL)},
			},
		}))

		defer app.Close()

		reloader := uncors.NewReloader(app, container.CliOutput(), func() (*config.UncorsConfig, error) {
			return &config.UncorsConfig{
				Mappings: config.Mappings{
					{From: hosts.Loopback.HTTPPort(port), To: hosts.Parse(second.URL)},
				},
			}, nil
		}, "")

		require.NoError(t, reloader.Reload(t.Context()))
		assert.Equal(t, "second", get(t, hosts.Loopback.HTTPPort(port).String()))
	})
}

func TestReloaderStart(t *testing.T) {
	t.Run("does nothing without a config file", func(t *testing.T) {
		reloader := uncors.NewReloader(nil, nil, nil, "")

		require.NoError(t, reloader.Start(t.Context()))
		assert.False(t, reloader.Watching())
		require.NoError(t, reloader.Close())
	})

	t.Run("fails for a missing config file", func(t *testing.T) {
		reloader := uncors.NewReloader(nil, nil, nil, "/no/such/config.yaml")

		require.Error(t, reloader.Start(t.Context()))
		assert.False(t, reloader.Watching())
	})

	t.Run("reloads on config file change", func(t *testing.T) {
		container := di.NewContainer(di.WithVersion(version))
		defer testutils.Close(t, container)

		configPath := filepath.Join(t.TempDir(), "config.yaml")
		require.NoError(t, os.WriteFile(configPath, []byte("proxy: \"\""), 0o600))

		reloaded := make(chan struct{}, 1)

		app := uncors.CreateUncors(container)
		reloader := uncors.NewReloader(app, container.CliOutput(), func() (*config.UncorsConfig, error) {
			select {
			case reloaded <- struct{}{}:
			default:
			}

			return &config.UncorsConfig{Mappings: config.Mappings{}}, nil
		}, configPath)

		require.NoError(t, reloader.Start(t.Context()))
		assert.True(t, reloader.Watching())

		defer testutils.Close(t, reloader)

		require.NoError(t, os.WriteFile(configPath, []byte("proxy: \"\"\n"), 0o600))

		select {
		case <-reloaded:
		case <-time.After(time.Second):
			t.Fatal("config change did not trigger a reload")
		}
	})

	t.Run("close is idempotent and stops watching", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		require.NoError(t, os.WriteFile(configPath, []byte("proxy: \"\""), 0o600))

		reloader := uncors.NewReloader(nil, nil, nil, configPath)

		require.NoError(t, reloader.Start(t.Context()))
		require.NoError(t, reloader.Close())
		require.NoError(t, reloader.Close())
		assert.False(t, reloader.Watching())
	})
}

func get(t *testing.T, url string) string {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	return string(body)
}
