package config_test

import (
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestStringToTimeDurationHookFunc(t *testing.T) {
	const key = "duration"
	viperInstance := viper.New()
	configOption := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		config.StringToTimeDurationHookFunc(),
		mapstructure.OrComposeDecodeHookFunc(
			mapstructure.StringToSliceHookFunc(","),
			mapstructure.StringToSliceHookFunc(", "),
		),
	))

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
				viperInstance.Set(key, testCase.value)

				durationValue := time.Duration(0)
				err := viperInstance.UnmarshalKey(key, &durationValue, configOption)
				testutils.CheckNoError(t, err)

				assert.Equal(t, testCase.expected, durationValue)
			})
		}
	})

	t.Run("doesnt not affected other type parses", func(t *testing.T) {
		t.Run("string to string", func(t *testing.T) {
			viperInstance.Set(key, "value")

			stringValue := ""
			err := viperInstance.UnmarshalKey(key, &stringValue, configOption)
			testutils.CheckNoError(t, err)

			assert.Equal(t, "value", stringValue)
		})

		t.Run("string to []string", func(t *testing.T) {
			viperInstance.Set(key, "value,value2")

			var stringValue []string
			err := viperInstance.UnmarshalKey(key, &stringValue, configOption)
			testutils.CheckNoError(t, err)

			assert.Equal(t, []string{"value", "value2"}, stringValue)
		})

		t.Run("number to string", func(t *testing.T) {
			viperInstance.Set(key, 11)

			stringValue := ""
			err := viperInstance.UnmarshalKey(key, &stringValue, configOption)
			testutils.CheckNoError(t, err)

			assert.Equal(t, "11", stringValue)
		})

		t.Run("number to duration", func(t *testing.T) {
			const expected = 14 * time.Minute
			viperInstance.Set(key, int(expected))

			durationValue := time.Nanosecond
			err := viperInstance.UnmarshalKey(key, &durationValue, configOption)
			testutils.CheckNoError(t, err)

			assert.Equal(t, expected, durationValue)
		})
	})
}
