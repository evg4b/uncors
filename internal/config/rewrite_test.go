package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewritingOptionClone(t *testing.T) {
	tests := []struct {
		name     string
		expected config.RewritingOption
	}{
		{
			name:     "empty structure",
			expected: config.RewritingOption{},
		},
		{
			name: "structure with 1 field",
			expected: config.RewritingOption{
				From: "from",
			},
		},
		{
			name: "structure with 2 fields",
			expected: config.RewritingOption{
				From: "from",
				To:   "to",
			},
		},
		{
			name: "structure with all fields",
			expected: config.RewritingOption{
				From: "from",
				To:   "to",
				Host: "host",
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

func TestRewriteOptionsClone(t *testing.T) {
	tests := []struct {
		name     string
		expected config.RewriteOptions
	}{
		{
			name:     "empty slice",
			expected: config.RewriteOptions{},
		},
		{
			name: "slice with one element",
			expected: config.RewriteOptions{
				{From: "from1", To: "to1", Host: "host1"},
			},
		},
		{
			name: "slice with multiple elements",
			expected: config.RewriteOptions{
				{From: "from1", To: "to1", Host: "host1"},
				{From: "from2", To: "to2", Host: "host2"},
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

const (
	fromPath = "/from/path"
	toPath   = "/to/path"
)

func TestRewritingOptionValidatorIsValidNoError(t *testing.T) {
	tests := []struct {
		name  string
		value config.RewritingOption
	}{
		{name: "valid paths and host", value: config.RewritingOption{From: fromPath, To: toPath, Host: hosts.Github.Host()}},
		{name: "no host", value: config.RewritingOption{From: fromPath, To: toPath}},
		{
			name:  "relative from path",
			value: config.RewritingOption{From: "../relative/from/path", To: toPath, Host: hosts.Github.Host()},
		},
		{
			name:  "relative to path",
			value: config.RewritingOption{From: fromPath, To: "../relative/to/path", Host: hosts.Github.Host()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, tt.value.Validate("testField"))
		})
	}
}

func TestRewritingOptionValidatorIsValidWithError(t *testing.T) {
	tests := []struct {
		name  string
		value config.RewritingOption
		error string
	}{
		{
			name:  "invalid from path",
			value: config.RewritingOption{From: "", To: toPath, Host: hosts.Github.Host()},
			error: "testField.from must not be empty",
		},
		{
			name:  "invalid to path",
			value: config.RewritingOption{From: fromPath, To: "", Host: hosts.Github.Host()},
			error: "testField.to must not be empty",
		},
		{
			name:  "invalid host format",
			value: config.RewritingOption{From: fromPath, To: toPath, Host: "exa mple.com"},
			error: "testField.host is not a valid host",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			require.EqualError(t, testCase.value.Validate("testField"), testCase.error)
		})
	}
}
