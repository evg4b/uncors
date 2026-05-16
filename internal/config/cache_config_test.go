package config_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCacheConfigUnmarshalYAML(t *testing.T) {
	t.Run("decodes all fields", func(t *testing.T) {
		const input = `
expiration-time: 30m
max-size: 52428800
methods:
  - GET
  - POST
`

		var actual config.CacheConfig

		require.NoError(t, yaml.Unmarshal([]byte(input), &actual))
		assert.Equal(t, config.CacheConfig{
			ExpirationTime: 30 * time.Minute,
			MaxSize:        52428800,
			Methods:        []string{http.MethodGet, http.MethodPost},
		}, actual)
	})

	t.Run("parses expiration-time with embedded spaces", func(t *testing.T) {
		const input = `expiration-time: "1h 30m"`

		var actual config.CacheConfig

		require.NoError(t, yaml.Unmarshal([]byte(input), &actual))
		assert.Equal(t, 90*time.Minute, actual.ExpirationTime)
	})

	t.Run("absent fields keep zero values", func(t *testing.T) {
		const input = `max-size: 1024`

		var actual config.CacheConfig

		require.NoError(t, yaml.Unmarshal([]byte(input), &actual))
		assert.Equal(t, int64(1024), actual.MaxSize)
		assert.Zero(t, actual.ExpirationTime)
		assert.Nil(t, actual.Methods)
	})

	t.Run("returns ErrInvalidCacheConfig for non-mapping node", func(t *testing.T) {
		const input = `- item1`

		var actual config.CacheConfig

		err := yaml.Unmarshal([]byte(input), &actual)

		assert.ErrorIs(t, err, config.ErrInvalidCacheConfig)
	})

	t.Run("returns error for invalid expiration-time", func(t *testing.T) {
		const input = `expiration-time: not-a-duration`

		var actual config.CacheConfig

		assert.Error(t, yaml.Unmarshal([]byte(input), &actual))
	})
}

func TestCacheGlobsClone(t *testing.T) {
	globs := config.CacheGlobs{
		"/api/**",
		"/constants",
		"/translations",
		"/**/*.js",
	}

	cacheGlobs := globs.Clone()

	t.Run("not same", func(t *testing.T) {
		assert.NotSame(t, &globs, &cacheGlobs)
	})

	t.Run("equals values", func(t *testing.T) {
		assert.Equal(t, globs, cacheGlobs)
	})
}

func TestCacheConfigClone(t *testing.T) {
	cacheConfig := &config.CacheConfig{
		ExpirationTime: 5 * time.Minute,
		MaxSize:        50 * 1024 * 1024,
		Methods:        []string{http.MethodGet, http.MethodPost},
	}

	clonedCacheConfig := cacheConfig.Clone()

	t.Run("not same", func(t *testing.T) {
		assert.NotSame(t, cacheConfig, clonedCacheConfig)
	})

	t.Run("equals values", func(t *testing.T) {
		assert.Equal(t, cacheConfig, clonedCacheConfig)
	})

	t.Run("not same methods", func(t *testing.T) {
		assert.NotSame(t, &cacheConfig.Methods, &clonedCacheConfig.Methods)
	})
}
