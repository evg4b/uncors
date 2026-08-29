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
			expected config.Duration
		}{
			{
				name:     "duration without spaces",
				input:    "expiration-time: 3h6m13s",
				expected: config.Duration(3*time.Hour + 6*time.Minute + 13*time.Second),
			},
			// The documentation writes multi-unit durations with spaces, so the
			// spelling a reader copies has to be the spelling uncors accepts.
			{
				name:     "duration with spaces",
				input:    "expiration-time: 1h 30m",
				expected: config.Duration(time.Hour + 30*time.Minute),
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

	t.Run("returns error for invalid duration string", func(t *testing.T) {
		var cfg config.CacheConfig

		err := yaml.Unmarshal([]byte("expiration-time: notaduration"), &cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not a valid duration")
		assert.Contains(t, err.Error(), "1m30s", "the error should show what a valid value looks like")
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
			expected config.Duration
		}{
			{
				name:     "millisecond delay",
				input:    "delay: 200ms",
				expected: config.Duration(200 * time.Millisecond),
			},
			{
				name:     "an hour with 500 milliseconds",
				input:    "delay: \"1h500ms\"",
				expected: config.Duration(time.Hour + 500*time.Millisecond),
			},
			{
				name:     "the documented spaced spellings",
				input:    "delay: 1m 30s",
				expected: config.Duration(time.Minute + 30*time.Second),
			},
			{
				name:     "milliseconds with a space",
				input:    "delay: 2s 500ms",
				expected: config.Duration(2*time.Second + 500*time.Millisecond),
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
		assert.Contains(t, err.Error(), "is not a valid duration")
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
