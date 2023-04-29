package configuration_test

import (
	"github.com/evg4b/uncors/internal/configuration"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

type test struct {
	Mappings []configuration.URLMapping `mapstructure:"mappings"`
}

func TestURLMappingHookFunc(t *testing.T) {
	tests := []struct {
		name string
		demo string
		want mapstructure.DecodeHookFunc
	}{
		{
			name: "",
			demo: `
mappings:
  - http://demo: https://demo.com
  - from: http://gffdgfdg
    to: https://gdfgfdg.com
  - from: http://de123mo.2123.com
    to: https://desafdmo.com
    mocks: ['1','2','3']
`,
			want: nil,
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			configFile := filepath.Join(t.TempDir(), "config.yaml")
			err := os.WriteFile(configFile, []byte(tt.demo), os.ModePerm)
			testutils.CheckNoError(t, err)

			viperInstance := viper.GetViper()
			viperInstance.SetConfigFile(configFile)
			err = viperInstance.ReadInConfig()
			testutils.CheckNoError(t, err)

			options := viper.DecodeHook(configuration.URLMappingHookFunc())

			d := test{}

			err = viperInstance.Unmarshal(&d, options)
			testutils.CheckNoError(t, err)

			assert.Equal(t, d, test{
				Mappings: []configuration.URLMapping{
					{From: "http://demo", To: "https://demo.com", Mocks: []string{}},
					{From: "http://gffdgfdg", To: "https://gdfgfdg.com", Mocks: []string{}},
					{From: "http://de123mo.2123.com", To: "https://desafdmo.com", Mocks: []string{"1", "2", "3"}},
				},
			})
		})
	}
}
