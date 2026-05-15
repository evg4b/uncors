package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var localhostSecure = "https://localhost:9090"

// decodeYAMLInto decodes a YAML string into out using mapstructure with the given hooks.
func decodeYAMLInto(t *testing.T, yamlStr string, out any, hooks ...mapstructure.DecodeHookFunc) {
	t.Helper()

	var raw any
	require.NoError(t, yaml.Unmarshal([]byte(yamlStr), &raw))

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           out,
		WeaklyTypedInput: true,
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(hooks...),
	})
	require.NoError(t, err)
	require.NoError(t, decoder.Decode(raw))
}

func TestURLMappingHookFunc(t *testing.T) {
	t.Run("positive cases", func(t *testing.T) {
		tests := []struct {
			name     string
			yaml     string
			expected config.Mapping
		}{
			{
				name: "simple key-value mapping",
				yaml: "http://localhost:4200: https://github.com",
				expected: config.Mapping{
					From: hosts.Localhost.HTTPPort(4200),
					To:   hosts.Github.HTTPS(),
				},
			},
			{
				name: "full object mapping",
				yaml: "{ from: http://localhost:3000, to: https://api.github.com }",
				expected: config.Mapping{
					From: hosts.Localhost.HTTPPort(3000),
					To:   hosts.APIGithub.HTTPS(),
				},
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				actual := config.Mapping{}
				decodeYAMLInto(t, testCase.yaml, &actual, config.URLMappingHookFunc())

				assert.Equal(t, testCase.expected, actual)
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
				From: hosts.Localhost.HTTP(),
			},
		},
		{
			name: "structure with 2 field",
			expected: config.Mapping{
				From: hosts.Localhost.HTTP(),
				To:   localhostSecure,
			},
		},
		{
			name: "structure with inner collections",
			expected: config.Mapping{
				From: hosts.Localhost.HTTP(),
				To:   localhostSecure,
				Statics: []config.StaticDirectory{
					{Path: "/cc", Dir: "cc"},
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			actual := testCase.expected.Clone()

			assert.NotSame(t, &testCase.expected, &actual)
			assert.Equal(t, testCase.expected, actual)
			assert.NotSame(t, &testCase.expected.Statics, &actual.Statics)
		})
	}
}
