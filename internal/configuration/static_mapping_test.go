package configuration_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/configuration"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestStaticDirMappingHookFunc(t *testing.T) {
	const configFile = "config.yaml"
	type testType struct {
		Statics configuration.StaticDirMappings `mapstructure:"statics"`
	}

	tests := []struct {
		name     string
		config   string
		expected configuration.StaticDirMappings
	}{
		{
			name: "decode plan mapping",
			config: `
statics:
  /path: /static-dir
  /another-path: /another-static-dir
`,
			expected: configuration.StaticDirMappings{
				{Path: "/another-path", Dir: "/another-static-dir"},
				{Path: "/path", Dir: "/static-dir"},
			},
		},
		{
			name: "decode object mappings",
			config: `
statics:
  /path: { dir: /static-dir }
  /another-path: { dir: /another-static-dir }
`,
			expected: configuration.StaticDirMappings{
				{Path: "/path", Dir: "/static-dir"},
				{Path: "/another-path", Dir: "/another-static-dir"},
			},
		},
		{
			name: "decode object mappings with default",
			config: `
statics:
  /path: { dir: /static-dir, default: index.html }
  /another-path: { dir: /another-static-dir, default: default.html }
`,
			expected: configuration.StaticDirMappings{
				{Path: "/path", Dir: "/static-dir", Default: "index.html"},
				{Path: "/another-path", Dir: "/another-static-dir", Default: "default.html"},
			},
		},
		{
			name: "decode mixed mappings with default",
			config: `
statics:
  /path: { dir: /static-dir, default: index.html }
  /another-path: /another-static-dir
`,
			expected: configuration.StaticDirMappings{
				{Path: "/path", Dir: "/static-dir", Default: "index.html"},
				{Path: "/another-path", Dir: "/another-static-dir"},
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

			actual := testType{}

			err = viperInstance.Unmarshal(&actual, viper.DecodeHook(
				configuration.StaticDirMappingHookFunc(),
			))
			testutils.CheckNoError(t, err)

			assert.ElementsMatch(t, actual.Statics, testCase.expected)
		})
	}
}
