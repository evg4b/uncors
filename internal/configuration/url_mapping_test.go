package configuration_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/configuration"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestURLMappingHookFunc(t *testing.T) {
	const configFile = "config.yaml"

	t.Run("positive cases", func(t *testing.T) {
		tests := []struct {
			name     string
			config   string
			expected configuration.URLMapping
		}{
			{
				name:   "simple key-value mapping",
				config: "http://localhost:4200: https://github.com",
				expected: configuration.URLMapping{
					From: "http://localhost:4200",
					To:   "https://github.com",
				},
			},
			{
				name:   "full object mapping",
				config: "{ from: http://localhost:3000, to: https://google.com }",
				expected: configuration.URLMapping{
					From: "http://localhost:3000",
					To:   "https://google.com",
				},
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				viperInstance := viper.GetViper()
				viperInstance.SetFs(testutils.FsFromMap(t, map[string]string{
					configFile: testCase.config,
				}))
				viperInstance.SetConfigFile(configFile)
				err := viperInstance.ReadInConfig()
				testutils.CheckNoError(t, err)

				actual := configuration.URLMapping{}

				err = viperInstance.Unmarshal(&actual, viper.DecodeHook(
					configuration.URLMappingHookFunc(),
				))
				testutils.CheckNoError(t, err)

				assert.Equal(t, actual, testCase.expected)
			})
		}
	})
}
