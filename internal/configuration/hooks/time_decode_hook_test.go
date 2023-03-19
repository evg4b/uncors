package hooks_test

import (
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/configuration/hooks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestStringToTimeDurationHookFunc(t *testing.T) {
	const key = "duration"
	viperInstance := viper.New()
	configOption := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		hooks.StringToTimeDurationHookFunc(),
	))

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
}
