package configuration

import (
	"fmt"

	"github.com/evg4b/uncors/internal/middlewares/mock"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	defaultHTTPPort  = 80
	defaultHTTPSPort = 443
)

type UncorsConfig struct {
	// Base config_test_data
	HTTPPort int               `mapstructure:"http-port" validate:"required"`
	Mappings map[string]string `mapstructure:"mappings" validate:"required"`
	Proxy    string            `mapstructure:"proxy"`
	Debug    bool              `mapstructure:"debug"`
	// HTTPS config_test_data
	HTTPSPort int    `mapstructure:"https-port"`
	CertFile  string `mapstructure:"cert-file"`
	KeyFile   string `mapstructure:"key-file"`
	// Mocks config_test_data
	Mocks []mock.Mock `mapstructure:"mocks"`
}

func (config *UncorsConfig) IsHTTPSEnabled() bool {
	return len(config.CertFile) > 0 && len(config.KeyFile) > 0 && config.HTTPSPort > 0
}

func LoadConfiguration(viperInstance *viper.Viper, args []string) (*UncorsConfig, error) {
	flags := defineFlags()
	if err := flags.Parse(args); err != nil {
		return nil, fmt.Errorf("filed parsing flags: %w", err)
	}

	if err := viperInstance.BindPFlags(flags); err != nil {
		return nil, fmt.Errorf("filed binding flags: %w", err)
	}

	configuration := &UncorsConfig{
		Mappings: map[string]string{},
		Mocks:    []mock.Mock{},
	}

	configPath := viperInstance.GetString("config")
	if len(configPath) > 0 {
		viperInstance.SetConfigFile(configPath)
		if err := viperInstance.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("filed to read config file '%s': %w", configPath, err)
		}
	}

	if err := viperInstance.Unmarshal(configuration); err != nil {
		return nil, fmt.Errorf("filed parsing configuraion: %w", err)
	}

	if err := readURLMapping(viperInstance, configuration); err != nil {
		return nil, fmt.Errorf("recognize url mapping: %w", err)
	}

	return configuration, nil
}

func defineFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("uncors", pflag.ContinueOnError)
	flags.Usage = pflag.Usage
	flags.StringSlice("to", []string{}, "Target host with protocol for to the resource to be proxy")
	flags.StringSlice("from", []string{}, "Local host with protocol for to the resource from which proxying will take place") //nolint: lll
	flags.UintP("http-port", "p", defaultHTTPPort, "Local HTTP listening port")
	flags.UintP("https-port", "s", defaultHTTPSPort, "Local HTTPS listening port")
	flags.String("cert-file", "", "Path to HTTPS certificate file")
	flags.String("key-file", "", "Path to matching for certificate private key")
	flags.String("proxy", "", "HTTP/HTTPS proxy to provide requests to real server (used system by default)")
	flags.Bool("debug", false, "Show debug output")
	flags.StringP("config", "c", "", "Show debug output")

	return flags
}
