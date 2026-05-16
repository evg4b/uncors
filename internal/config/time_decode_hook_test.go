package config_test

import (
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCacheConfigDurationUnmarshal(t *testing.T) {
	t.Run("parses valid duration strings", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected time.Duration
		}{
			{
				name:     "duration without spaces",
				input:    "expiration-time: 3h6m13s",
				expected: 3*time.Hour + 6*time.Minute + 13*time.Second,
			},
			{
				name:     "duration with spaces",
				input:    "expiration-time: \"1m 4s\"",
				expected: 1*time.Minute + 4*time.Second,
			},
			{
				name:     "duration with mixed spaces",
				input:    "expiration-time: \"1h 3m59s 40ms\"",
				expected: 1*time.Hour + 3*time.Minute + 59*time.Second + 40*time.Millisecond,
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				cfg := config.CacheConfig{ExpirationTime: config.DefaultExpirationTime}
				require.NoError(t, yaml.Unmarshal([]byte(testCase.input), &cfg))
				assert.Equal(t, testCase.expected, cfg.ExpirationTime)
			})
		}
	})

	t.Run("preserves defaults for absent fields", func(t *testing.T) {
		cfg := config.CacheConfig{
			ExpirationTime: config.DefaultExpirationTime,
			MaxSize:        config.DefaultMaxSize,
			Methods:        []string{"GET"},
		}

		require.NoError(t, yaml.Unmarshal([]byte("max-size: 1048576"), &cfg))
		assert.Equal(t, config.DefaultExpirationTime, cfg.ExpirationTime)
		assert.Equal(t, int64(1048576), cfg.MaxSize)
	})

	t.Run("returns error for non-mapping input", func(t *testing.T) {
		var cfg config.CacheConfig

		err := yaml.Unmarshal([]byte("just-a-string"), &cfg)
		assert.ErrorIs(t, err, config.ErrInvalidCacheConfig)
	})

	t.Run("returns error for invalid duration string", func(t *testing.T) {
		var cfg config.CacheConfig

		err := yaml.Unmarshal([]byte("expiration-time: notaduration"), &cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid expiration-time")
	})

	t.Run("returns error when max-size is not a number", func(t *testing.T) {
		var cfg config.CacheConfig

		err := yaml.Unmarshal([]byte("max-size: [a, b, c]"), &cfg)
		require.Error(t, err)
	})

	t.Run("returns error when methods is not a sequence", func(t *testing.T) {
		var cfg config.CacheConfig

		err := yaml.Unmarshal([]byte("methods: {key: value}"), &cfg)
		require.Error(t, err)
	})
}

func TestResponseDelayUnmarshal(t *testing.T) {
	t.Run("parses valid delay strings", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected time.Duration
		}{
			{
				name:     "millisecond delay",
				input:    "delay: 200ms",
				expected: 200 * time.Millisecond,
			},
			{
				name:     "delay with spaces",
				input:    "delay: \"1s 500ms\"",
				expected: 1*time.Second + 500*time.Millisecond,
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				var resp config.Response
				require.NoError(t, yaml.Unmarshal([]byte(testCase.input), &resp))
				assert.Equal(t, testCase.expected, resp.Delay)
			})
		}
	})

	t.Run("returns error for invalid delay string", func(t *testing.T) {
		var resp config.Response

		err := yaml.Unmarshal([]byte("delay: notaduration"), &resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid delay")
	})

	t.Run("zero delay when field absent", func(t *testing.T) {
		var resp config.Response

		require.NoError(t, yaml.Unmarshal([]byte("code: 200"), &resp))
		assert.Zero(t, resp.Delay)
	})

	t.Run("returns error when response is not a mapping", func(t *testing.T) {
		var resp config.Response

		err := yaml.Unmarshal([]byte("[200, 404]"), &resp)
		require.Error(t, err)
	})
}
