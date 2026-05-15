package config_test

import (
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodeValue uses mapstructure with the given hooks to decode src into dst.
func decodeValue(t *testing.T, src, dst any, hooks ...mapstructure.DecodeHookFunc) error {
	t.Helper()

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           dst,
		WeaklyTypedInput: true,
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(hooks...),
	})
	require.NoError(t, err)

	return decoder.Decode(src)
}

func TestStringToTimeDurationHookFunc(t *testing.T) {
	hook := config.StringToTimeDurationHookFunc()
	sliceHook := mapstructure.StringToSliceHookFunc(",")

	t.Run("correct parse different formats", func(t *testing.T) {
		tests := []struct {
			name     string
			value    string
			expected time.Duration
		}{
			{
				name:     "duration with spaces",
				value:    "1m 4s",
				expected: 1*time.Minute + 4*time.Second,
			},
			{
				name:     "duration without spaces",
				value:    "3h6m13s",
				expected: 3*time.Hour + 6*time.Minute + 13*time.Second,
			},
			{
				name:     "duration with mixed spaces",
				value:    "1h 3m59s 40ms",
				expected: 1*time.Hour + 3*time.Minute + 59*time.Second + 40*time.Millisecond,
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				var out time.Duration
				require.NoError(t, decodeValue(t, testCase.value, &out, hook))
				assert.Equal(t, testCase.expected, out)
			})
		}
	})

	t.Run("doesnt not affected other type parses", func(t *testing.T) {
		t.Run("string to string", func(t *testing.T) {
			var out string
			require.NoError(t, decodeValue(t, "value", &out, hook))
			assert.Equal(t, "value", out)
		})

		t.Run("string to []string", func(t *testing.T) {
			var out []string
			require.NoError(t, decodeValue(t, "value,value2", &out, sliceHook))
			assert.Equal(t, []string{"value", "value2"}, out)
		})

		t.Run("number to string", func(t *testing.T) {
			var out string
			require.NoError(t, decodeValue(t, 11, &out, hook))
			assert.Equal(t, "11", out)
		})

		t.Run("number to duration", func(t *testing.T) {
			const expected = 14 * time.Minute

			out := time.Nanosecond
			require.NoError(t, decodeValue(t, int(expected), &out, hook))
			assert.Equal(t, expected, out)
		})
	})
}
