package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var (
	localhost       = "http://localhost"
	localhostSecure = "https://localhost:9090"
)

func TestURLMappingHookFunc(t *testing.T) {
	const configFile = "config.yaml"

	t.Run("positive cases", func(t *testing.T) {
		tests := []struct {
			name     string
			config   string
			expected config.Mapping
		}{
			{
				name:   "simple key-value mapping",
				config: "http://localhost:4200: https://github.com",
				expected: config.Mapping{
					From: "http://localhost:4200",
					To:   "https://github.com",
				},
			},
			{
				name:   "full object mapping",
				config: "{ from: http://localhost:3000, to: https://google.com }",
				expected: config.Mapping{
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

				actual := config.Mapping{}

				err = viperInstance.Unmarshal(&actual, viper.DecodeHook(
					config.URLMappingHookFunc(),
				))
				testutils.CheckNoError(t, err)

				assert.Equal(t, actual, testCase.expected)
			})
		}
	})
}

func TestURLMappingClone(t *testing.T) {
	tests := []struct {
		name     string
		expected config.Mapping
	}{
		{
			name:     "empty structure",
			expected: config.Mapping{},
		},
		{
			name: "structure with 1 field",
			expected: config.Mapping{
				From: localhost,
			},
		},
		{
			name: "structure with 2 field",
			expected: config.Mapping{
				From: localhost,
				To:   localhostSecure,
			},
		},
		{
			name: "structure with inner collections",
			expected: config.Mapping{
				From: localhost,
				To:   localhostSecure,
				Statics: []config.StaticDir{
					{Path: "/cc", Dir: "cc"},
				},
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			actual := testCase.expected.Clone()

			assert.NotSame(t, testCase.expected, actual)
			assert.Equal(t, testCase.expected, actual)
			assert.NotSame(t, testCase.expected.Statics, actual.Statics)
		})
	}
}
