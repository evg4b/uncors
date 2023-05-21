package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

const (
	anotherStaticDir = "/another-static-dir"
	anotherPath      = "/another-path"
	path             = "/path"
	staticDir        = "/static-dir"
)

func TestStaticDirMappingHookFunc(t *testing.T) {
	const configFile = "config.yaml"
	type testType struct {
		Statics config.StaticDirMappings `mapstructure:"statics"`
	}

	tests := []struct {
		name     string
		config   string
		expected config.StaticDirMappings
	}{
		{
			name: "decode plan mapping",
			config: `
statics:
  /path: /static-dir
  /another-path: /another-static-dir
`,
			expected: config.StaticDirMappings{
				{Path: anotherPath, Dir: anotherStaticDir},
				{Path: path, Dir: staticDir},
			},
		},
		{
			name: "decode object mappings",
			config: `
statics:
  /path: { dir: /static-dir }
  /another-path: { dir: /another-static-dir }
`,
			expected: config.StaticDirMappings{
				{Path: path, Dir: staticDir},
				{Path: anotherPath, Dir: anotherStaticDir},
			},
		},
		{
			name: "decode object mappings with index",
			config: `
statics:
  /path: { dir: /static-dir, index: index.html }
  /another-path: { dir: /another-static-dir, index: default.html }
`,
			expected: config.StaticDirMappings{
				{Path: path, Dir: staticDir, Index: "index.html"},
				{Path: anotherPath, Dir: anotherStaticDir, Index: "default.html"},
			},
		},
		{
			name: "decode mixed mappings with index",
			config: `
statics:
  /path: { dir: /static-dir, index: index.html }
  /another-path: /another-static-dir
`,
			expected: config.StaticDirMappings{
				{Path: path, Dir: staticDir, Index: "index.html"},
				{Path: anotherPath, Dir: anotherStaticDir},
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
				config.StaticDirMappingHookFunc(),
			))
			testutils.CheckNoError(t, err)

			assert.ElementsMatch(t, actual.Statics, testCase.expected)
		})
	}
}

func TestStaticDirMappingClone(t *testing.T) {
	tests := []struct {
		name     string
		expected config.StaticDirMapping
	}{
		{
			name:     "empty structure",
			expected: config.StaticDirMapping{},
		},
		{
			name: "structure with 1 field",
			expected: config.StaticDirMapping{
				Dir: "dir",
			},
		},
		{
			name: "structure with 2 field",
			expected: config.StaticDirMapping{
				Dir:  "dir",
				Path: "/some-path",
			},
		},
		{
			name: "structure with all field",
			expected: config.StaticDirMapping{
				Dir:   "dir",
				Path:  "/one-more-path",
				Index: "index.html",
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			actual := testCase.expected.Clone()

			assert.NotSame(t, testCase.expected, actual)
			assert.Equal(t, testCase.expected, actual)
		})
	}
}
