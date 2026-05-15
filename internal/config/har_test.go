package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHARConfigHookFunc(t *testing.T) {
	decode := func(t *testing.T, raw any) config.HARConfig {
		t.Helper()

		var out config.HARConfig

		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Result:     &out,
			DecodeHook: config.HARConfigHookFunc(),
		})
		require.NoError(t, err)
		require.NoError(t, decoder.Decode(raw))

		return out
	}

	t.Run("string shorthand sets File", func(t *testing.T) {
		cfg := decode(t, "./recordings/api.har")
		assert.Equal(t, config.HARConfig{File: "./recordings/api.har"}, cfg)
	})

	t.Run("map form decoded normally", func(t *testing.T) {
		cfg := decode(t, map[string]any{
			"file":                   "./out.har",
			"capture-secure-headers": true,
		})
		assert.Equal(t, config.HARConfig{
			File:                 "./out.har",
			CaptureSecureHeaders: true,
		}, cfg)
	})
}

func TestHARShorthandInMapping(t *testing.T) {
	const configFile = "config.yaml"

	const yaml = `
from: http://localhost:3000
to: https://api.example.com
har: ./recordings/api.har
`

	viperCfg := viper.New()
	viperCfg.SetFs(testutils.FsFromMap(t, map[string]string{configFile: yaml}))
	viperCfg.SetConfigFile(configFile)
	require.NoError(t, viperCfg.ReadInConfig())

	actual := config.Mapping{}
	require.NoError(t, viperCfg.Unmarshal(&actual, viper.DecodeHook(
		config.URLMappingHookFunc(),
	)))

	assert.Equal(t, "./recordings/api.har", actual.HAR.File)
	assert.False(t, actual.HAR.CaptureSecureHeaders)
}
