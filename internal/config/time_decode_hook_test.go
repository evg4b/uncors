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
}

func TestResponseDelayUnmarshal(t *testing.T) {
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
}
