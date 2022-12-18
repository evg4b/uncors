package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/mocks"
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
			mocks.SourceHost1: mocks.TargetHost1,
			mocks.SourceHost2: mocks.TargetHost2,
			mocks.SourceHost3: mocks.TargetHost3,
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
				from:        []string{mocks.SourceHost1},
				to:          []string{},
				expectedErr: "`to` values are not set for every `from`",
			},
			{
				name:        "to is not set",
				from:        []string{},
				to:          []string{mocks.TargetHost1},
				expectedErr: "`from` values are not set for every `to`",
			},
			{
				name:        "count of from values great then count of to",
				from:        []string{mocks.SourceHost1, mocks.SourceHost2},
				to:          []string{mocks.TargetHost1},
				expectedErr: "`to` values are not set for every `from`",
			},
			{
				name:        "count of to values great then count of from",
				from:        []string{mocks.SourceHost1},
				to:          []string{mocks.TargetHost1, mocks.TargetHost2},
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
