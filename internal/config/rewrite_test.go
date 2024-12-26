package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/stretchr/testify/assert"
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
