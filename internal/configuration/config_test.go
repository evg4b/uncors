package configuration_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/configuration"
	"github.com/evg4b/uncors/internal/middlewares/mock"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfiguration(t *testing.T) {
	fs := testutils.CreateFsForTest(t, "config_test_data")
	viperInstance := viper.New()
	viperInstance.SetFs(fs)

	t.Run("return default config", func(t *testing.T) {
		config, err := configuration.LoadConfiguration(viperInstance, []string{})
		assert.NoError(t, err)
		assert.Equal(t, &configuration.UncorsConfig{
			HTTPPort:  80,
			HTTPSPort: 443,
			Mappings:  map[string]string{},
			Mocks:     []mock.Mock{},
		}, config)
	})

	t.Run("correctly parse configuration", func(t *testing.T) {
		tests := []struct {
			name   string
			args   []string
			config *configuration.UncorsConfig
		}{
			{
				name: "minimal config is set",
				args: []string{"--config", "/minimal-config.yaml"},
				config: &configuration.UncorsConfig{
					HTTPPort:  8080,
					HTTPSPort: 443,
					Mappings: map[string]string{
						"http://demo": "https://demo.com",
					},
					Mocks: []mock.Mock{},
				},
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				config, err := configuration.LoadConfiguration(viperInstance, testCase.args)

				assert.NoError(t, err)
				assert.Equal(t, testCase.config, config)
			})
		}
	})
}
