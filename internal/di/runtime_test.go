package di_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func harConfiguration() *config.UncorsConfig {
	return &config.UncorsConfig{
		CacheConfig: config.CacheConfig{MaxSize: 100, ExpirationTime: time.Minute},
		Mappings: config.Mappings{
			{
				From:  hosts.Localhost.HTTP(),
				To:    hosts.Localhost.HTTPS(),
				Cache: config.CacheGlobs{"*.json"},
				HAR:   config.HARConfig{File: "/test.har"},
			},
		},
	}
}

func TestBuildRuntime(t *testing.T) {
	t.Run("builds a target per port group", func(t *testing.T) {
		container := di.NewContainer(di.WithFs(afero.NewMemMapFs()))
		defer testutils.Close(t, container)

		runtime, err := container.BuildRuntime(harConfiguration())
		require.NoError(t, err)

		defer testutils.Close(t, runtime)

		require.Len(t, runtime.Targets(), 1)
		assert.NotNil(t, runtime.Targets()[0].Handler)
	})

	t.Run("cache middleware uses the current cache config", func(t *testing.T) {
		container := di.NewContainer(di.WithFs(afero.NewMemMapFs()))
		defer testutils.Close(t, container)

		runtime, err := container.BuildRuntime(harConfiguration())
		require.NoError(t, err)

		defer testutils.Close(t, runtime)

		assert.Implements(t, (*contracts.Middleware)(nil), runtime.CacheMiddleware(config.CacheGlobs{"*.json"}))

		store := runtime.Cache()

		assert.Same(t, store, runtime.Cache())
	})

	t.Run("every generation gets its own resources", func(t *testing.T) {
		container := di.NewContainer(di.WithFs(afero.NewMemMapFs()))
		defer testutils.Close(t, container)

		first, err := container.BuildRuntime(harConfiguration())
		require.NoError(t, err)

		second, err := container.BuildRuntime(harConfiguration())
		require.NoError(t, err)

		assert.NotSame(t, first.Cache(), second.Cache())

		require.NoError(t, first.Close())
		require.NoError(t, second.Close())
	})

	t.Run("close is idempotent", func(t *testing.T) {
		container := di.NewContainer(di.WithFs(afero.NewMemMapFs()))
		defer testutils.Close(t, container)

		instance, err := container.BuildRuntime(harConfiguration())
		require.NoError(t, err)

		require.NoError(t, instance.Close())
		require.NoError(t, instance.Close())
	})

	// Reloading the config used to leak a HAR writer, its goroutine and a
	// ristretto instance per reload, with several writers racing over the same
	// file. Generation scoped resources make the growth bounded.
	t.Run("repeated reloads do not leak goroutines", func(t *testing.T) {
		const reloads = 50

		container := di.NewContainer(di.WithFs(afero.NewMemMapFs()))
		defer testutils.Close(t, container)

		warmUp, err := container.BuildRuntime(harConfiguration())
		require.NoError(t, err)
		require.NoError(t, warmUp.Close())

		baseline := runtime.NumGoroutine()

		for range reloads {
			generation, buildErr := container.BuildRuntime(harConfiguration())
			require.NoError(t, buildErr)
			require.NoError(t, generation.Close())
		}

		assert.LessOrEqual(t, runtime.NumGoroutine(), baseline+reloads/2,
			"config reloads must not accumulate goroutines")
	})
}
