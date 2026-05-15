package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var localhostSecure = "https://localhost:9090"

func TestMappingUnmarshalYAML(t *testing.T) {
	t.Run("positive cases", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected config.Mapping
		}{
			{
				name:  "simple key-value shorthand",
				input: "http://localhost:4200: https://github.com",
				expected: config.Mapping{
					From: hosts.Localhost.HTTPPort(4200),
					To:   hosts.Github.HTTPS(),
				},
			},
			{
				name:  "full object mapping",
				input: "{ from: http://localhost:3000, to: https://api.github.com }",
				expected: config.Mapping{
					From: hosts.Localhost.HTTPPort(3000),
					To:   hosts.APIGithub.HTTPS(),
				},
			},
			{
				name: "mapping with HAR shorthand",
				input: `
from: http://localhost:3000
to: https://api.example.com
har: ./recordings/api.har
`,
				expected: config.Mapping{
					From: hosts.Localhost.HTTPPort(3000),
					To:   "https://api.example.com",
					HAR:  config.HARConfig{File: "./recordings/api.har"},
				},
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				var actual config.Mapping
				require.NoError(t, yaml.Unmarshal([]byte(testCase.input), &actual))
				assert.Equal(t, testCase.expected, actual)
			})
		}
	})

	t.Run("error cases", func(t *testing.T) {
		t.Run("shorthand with non-string value", func(t *testing.T) {
			var actual config.Mapping

			err := yaml.Unmarshal([]byte("http://localhost: 123"), &actual)
			assert.Error(t, err)
		})
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
