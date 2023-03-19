package configuration_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/configuration"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		config   *configuration.UncorsConfig
		expected string
	}{
		{
			name: "invalid http-port",
			config: &configuration.UncorsConfig{
				Mappings: map[string]string{},
			},
			expected: "Key: 'UncorsConfig.HTTPPort' Error:Field validation for 'HTTPPort' failed on the 'required' tag",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := configuration.Validate(testCase.config)

			assert.EqualError(t, err, testCase.expected)
		})
	}
}
