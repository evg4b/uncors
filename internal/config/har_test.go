package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestHARConfigUnmarshalYAML(t *testing.T) {
	t.Run("string shorthand sets File", func(t *testing.T) {
		var cfg config.HARConfig
		require.NoError(t, yaml.Unmarshal([]byte(`"./recordings/api.har"`), &cfg))
		assert.Equal(t, config.HARConfig{File: "./recordings/api.har"}, cfg)
	})

	t.Run("map form decoded normally", func(t *testing.T) {
		const input = `
file: ./out.har
capture-secure-headers: true
`

		var cfg config.HARConfig
		require.NoError(t, yaml.Unmarshal([]byte(input), &cfg))
		assert.Equal(t, config.HARConfig{
			File:                 "./out.har",
			CaptureSecureHeaders: true,
		}, cfg)
	})
}

func TestHARShorthandInMapping(t *testing.T) {
	const input = `
from: http://localhost:3000
to: https://api.example.com
har: ./recordings/api.har
`

	var actual config.Mapping
	require.NoError(t, yaml.Unmarshal([]byte(input), &actual))

	assert.Equal(t, "./recordings/api.har", actual.HAR.File)
	assert.False(t, actual.HAR.CaptureSecureHeaders)
}
