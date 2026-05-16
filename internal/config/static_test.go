package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const (
	anotherStaticDir = "/another-static-dir"
	anotherPath      = "/another-path"
	path             = "/path"
	staticDir        = "/static-dir"
)

const (
	indexHTML = "index.html"
)

func TestStaticDirectoriesUnmarshalYAML(t *testing.T) {
	type testType struct {
		Statics config.StaticDirectories `yaml:"statics"`
	}

	t.Run("map form", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected config.StaticDirectories
		}{
			{
				name: "plain map shorthand",
				input: `
statics:
  /path: /static-dir
  /another-path: /another-static-dir
`,
				expected: config.StaticDirectories{
					{Path: path, Dir: staticDir},
					{Path: anotherPath, Dir: anotherStaticDir},
				},
			},
			{
				name: "object map without index",
				input: `
statics:
  /path: { dir: /static-dir }
  /another-path: { dir: /another-static-dir }
`,
				expected: config.StaticDirectories{
					{Path: path, Dir: staticDir},
					{Path: anotherPath, Dir: anotherStaticDir},
				},
			},
			{
				name: "object map with index",
				input: `
statics:
  /path: { dir: /static-dir, index: index.html }
  /another-path: { dir: /another-static-dir, index: default.html }
`,
				expected: config.StaticDirectories{
					{Path: path, Dir: staticDir, Index: indexHTML},
					{Path: anotherPath, Dir: anotherStaticDir, Index: "default.html"},
				},
			},
			{
				name: "mixed map",
				input: `
statics:
  /path: { dir: /static-dir, index: index.html }
  /another-path: /another-static-dir
`,
				expected: config.StaticDirectories{
					{Path: path, Dir: staticDir, Index: indexHTML},
					{Path: anotherPath, Dir: anotherStaticDir},
				},
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				var actual testType

				require.NoError(t, yaml.Unmarshal([]byte(testCase.input), &actual))
				assert.ElementsMatch(t, testCase.expected, actual.Statics)
			})
		}
	})

	t.Run("object map with invalid field type returns error", func(t *testing.T) {
		const input = `
statics:
  /path: [a, b, c]
`

		var actual testType

		assert.Error(t, yaml.Unmarshal([]byte(input), &actual))
	})

	t.Run("sequence form", func(t *testing.T) {
		const input = `
statics:
  - path: /path
    dir: /static-dir
  - path: /another-path
    dir: /another-static-dir
    index: index.html
`

		var actual testType

		require.NoError(t, yaml.Unmarshal([]byte(input), &actual))
		assert.Equal(t, config.StaticDirectories{
			{Path: path, Dir: staticDir},
			{Path: anotherPath, Dir: anotherStaticDir, Index: indexHTML},
		}, actual.Statics)
	})
}

func TestStaticDirMappingClone(t *testing.T) {
	tests := []struct {
		name     string
		expected config.StaticDirectory
	}{
		{
			name:     "empty structure",
			expected: config.StaticDirectory{},
		},
		{
			name: "structure with 1 field",
			expected: config.StaticDirectory{
				Dir: "dir",
			},
		},
		{
			name: "structure with 2 field",
			expected: config.StaticDirectory{
				Dir:  "dir",
				Path: "/some-path",
			},
		},
		{
			name: "structure with all field",
			expected: config.StaticDirectory{
				Dir:   "dir",
				Path:  "/one-more-path",
				Index: indexHTML,
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			actual := testCase.expected.Clone()

			assert.NotSame(t, &testCase.expected, &actual)
			assert.Equal(t, testCase.expected, actual)
		})
	}
}

func TestStaticValidator(t *testing.T) {
	const (
		assetsPath    = "/assets"
		staticPath    = "/static"
		indexFilePath = "/static/index.html"
	)

	fs := testutils.FsFromMap(t, map[string]string{indexFilePath: indexFilePath})

	t.Run("should not register errors if response is valid", func(t *testing.T) {
		tests := []struct {
			name  string
			value config.StaticDirectory
		}{
			{
				name:  "valid static directory with index",
				value: config.StaticDirectory{Path: assetsPath, Dir: staticPath, Index: "index.html"},
			},
			{
				name:  "valid static directory without index",
				value: config.StaticDirectory{Path: assetsPath, Dir: staticPath},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				assert.NoError(t, test.value.Validate("test", fs))
			})
		}
	})

	t.Run("should register errors if response is invalid", func(t *testing.T) {
		tests := []struct {
			name  string
			value config.StaticDirectory
			error string
		}{
			{
				name:  "empty path",
				value: config.StaticDirectory{Path: "", Dir: staticPath},
				error: "test.path must not be empty",
			},
			{
				name:  "empty directory",
				value: config.StaticDirectory{Path: assetsPath, Dir: ""},
				error: "test.directory must not be empty",
			},
			{
				name:  "missing index file",
				value: config.StaticDirectory{Path: assetsPath, Dir: staticPath, Index: "index.php"},
				error: "test.index /static/index.php does not exist",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				require.EqualError(t, test.value.Validate("test", fs), test.error)
			})
		}
	})
}
