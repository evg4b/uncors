package engine_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/engine"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
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

func TestBuildGeneration(t *testing.T) {
	t.Run("builds a target per port group", func(t *testing.T) {
		proxy := engine.New(engine.WithFs(afero.NewMemMapFs()))

		generation, err := proxy.BuildGeneration(harConfiguration())
		require.NoError(t, err)

		defer testutils.Close(t, generation)

		require.Len(t, generation.Targets(), 1)
		assert.NotNil(t, generation.Targets()[0].Handler)
	})

	t.Run("cache middleware uses the current cache config", func(t *testing.T) {
		proxy := engine.New(engine.WithFs(afero.NewMemMapFs()))

		generation, err := proxy.BuildGeneration(harConfiguration())
		require.NoError(t, err)

		defer testutils.Close(t, generation)

		assert.Implements(t, (*contracts.Middleware)(nil), generation.CacheMiddleware(config.CacheGlobs{"*.json"}))

		store := generation.Cache()

		assert.Same(t, store, generation.Cache())
	})

	t.Run("every generation gets its own resources", func(t *testing.T) {
		proxy := engine.New(engine.WithFs(afero.NewMemMapFs()))

		first, err := proxy.BuildGeneration(harConfiguration())
		require.NoError(t, err)

		second, err := proxy.BuildGeneration(harConfiguration())
		require.NoError(t, err)

		assert.NotSame(t, first.Cache(), second.Cache())

		require.NoError(t, first.Close())
		require.NoError(t, second.Close())
	})

	t.Run("close is idempotent", func(t *testing.T) {
		proxy := engine.New(engine.WithFs(afero.NewMemMapFs()))

		instance, err := proxy.BuildGeneration(harConfiguration())
		require.NoError(t, err)

		require.NoError(t, instance.Close())
		require.NoError(t, instance.Close())
	})

	t.Run("repeated reloads do not leak goroutines", func(t *testing.T) {
		const reloads = 50

		proxy := engine.New(engine.WithFs(afero.NewMemMapFs()))

		warmUp, err := proxy.BuildGeneration(harConfiguration())
		require.NoError(t, err)
		require.NoError(t, warmUp.Close())

		baseline := runtime.NumGoroutine()

		for range reloads {
			next, buildErr := proxy.BuildGeneration(harConfiguration())
			require.NoError(t, buildErr)
			require.NoError(t, next.Close())
		}

		assert.LessOrEqual(t, runtime.NumGoroutine(), baseline+reloads/2,
			"config reloads must not accumulate goroutines")
	})
}

func TestGenerationBuildsEveryHandler(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/www", 0o755))

	spans := testutils.Telemetry(t).Spans
	proxy := engine.New(engine.WithFs(fs))

	generation, err := proxy.BuildGeneration(&config.UncorsConfig{
		CacheConfig: config.CacheConfig{MaxSize: 100, ExpirationTime: time.Minute},
		Mappings: config.Mappings{{
			From:    hosts.Localhost.HTTP(),
			To:      hosts.Localhost.HTTPS(),
			Statics: config.StaticDirectories{{Path: "/assets", Dir: "/www"}},
			Mocks: config.Mocks{{
				Matcher:  config.RequestMatcher{Path: "/mock"},
				Response: config.Response{Code: 200, Raw: "mock"},
			}},
			Scripts: config.Scripts{{
				Matcher: config.RequestMatcher{Path: "/script"},
				Script:  `response:WriteHeader(200)`,
			}},
			Rewrites: config.RewriteOptions{{From: "/old", To: "/new"}},
		}},
	})
	require.NoError(t, err)

	defer testutils.Close(t, generation)

	require.Len(t, generation.Targets(), 1)
	assert.Implements(t, (*contracts.Middleware)(nil),
		generation.StaticMiddleware("/assets", config.StaticDirectory{Path: "/assets", Dir: "/www"}))
	assert.Implements(t, (*contracts.Middleware)(nil),
		generation.RewriteMiddleware(&config.RewritingOption{From: "/old", To: "/new"}))
	assert.Implements(t, (*contracts.Handler)(nil),
		generation.MockHandler(&config.Response{Code: 200, Raw: "mock"}))
	assert.Implements(t, (*contracts.Handler)(nil),
		generation.ScriptHandler(&config.Script{Script: `response:WriteHeader(200)`}))

	ended := spans.Ended()
	require.Len(t, ended, 1)
	assert.Equal(t, "engine.build_generation", ended[0].Name())
	assert.Contains(t, ended[0].Attributes(), attribute.Int("uncors.mappings", 1))
}
