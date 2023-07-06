package config_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestCacheGlobsClone(t *testing.T) {
	globs := config.CacheGlobs{
		"/api/**",
		"/constants",
		"/translations",
		"/**/*.js",
	}

	cacheGlobs := globs.Clone()

	t.Run("not same", func(t *testing.T) {
		assert.NotSame(t, globs, cacheGlobs)
	})

	t.Run("equals values", func(t *testing.T) {
		assert.EqualValues(t, globs, cacheGlobs)
	})
}

func TestCacheConfigClone(t *testing.T) {
	cacheConfig := &config.CacheConfig{
		ExpirationTime: 5 * time.Minute,
		ClearTime:      30 * time.Second,
		Methods:        []string{http.MethodGet, http.MethodPost},
	}

	clonedCacheConfig := cacheConfig.Clone()

	t.Run("not same", func(t *testing.T) {
		assert.NotSame(t, cacheConfig, clonedCacheConfig)
	})

	t.Run("equals values", func(t *testing.T) {
		assert.EqualValues(t, cacheConfig, clonedCacheConfig)
	})

	t.Run("not same methods", func(t *testing.T) {
		assert.NotSame(t, cacheConfig.Methods, clonedCacheConfig.Methods)
	})
}
