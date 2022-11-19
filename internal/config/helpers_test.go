package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestReadURLMapping(t *testing.T) {
	t.Run("correctly map pairs", func(t *testing.T) {
		viperConfig := viper.New()
		viperConfig.Set("from", []string{"host1", "host2", "host3"})
		viperConfig.Set("to", []string{"target-host1", "target-host2", "target-host3"})
		actual, err := config.ReadURLMapping(viperConfig)

		assert.NoError(t, err)
		assert.EqualValues(t, map[string]string{
			"host1": "target-host1",
			"host2": "target-host2",
			"host3": "target-host3",
		}, actual)
	})

	t.Run("incorrect pairs", func(t *testing.T) {
		tests := []struct {
			name        string
			from        []string
			to          []string
			expectedErr string
		}{
			{
				name:        "from is not set",
				from:        []string{"host1"},
				to:          []string{},
				expectedErr: "`to` values are not set for every `from`",
			},
			{
				name:        "to is not set",
				from:        []string{},
				to:          []string{"target-host1"},
				expectedErr: "`from` values are not set for every `to`",
			},
			{
				name:        "count of from values greath then count of to",
				from:        []string{"host1", "host2"},
				to:          []string{"target-host1"},
				expectedErr: "`to` values are not set for every `from`",
			},
			{
				name:        "count of to values greath then count of from",
				from:        []string{"host1"},
				to:          []string{"target-host1", "target-host2"},
				expectedErr: "`from` values are not set for every `to`",
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				viperConfig := viper.New()
				viperConfig.Set("from", testCase.from)
				viperConfig.Set("to", testCase.to)

				actual, err := config.ReadURLMapping(viperConfig)

				assert.Nil(t, actual)
				assert.EqualError(t, err, testCase.expectedErr)
			})
		}
	})
}
